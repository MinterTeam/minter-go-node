package swap

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
)

var Bound = big.NewInt(minimumLiquidity)

const minimumLiquidity = 1000
const commission = 2

type EditableChecker interface {
	// todo: add delete orders
	Exists() bool
	GetID() uint32
	// Deprecated
	AddLastSwapStep(amount0In, amount1Out *big.Int) EditableChecker
	AddLastSwapStepWithOrders(amount0In, amount1Out *big.Int) EditableChecker
	Reverse() EditableChecker
	Price() *big.Float
	Reserves() (reserve0 *big.Int, reserve1 *big.Int)
	Amounts(liquidity, totalSupply *big.Int) (amount0 *big.Int, amount1 *big.Int)
	CalculateAddAmountsForPrice(float *big.Float) (amount0, amount1 *big.Int)
	// CalculateAddAmountsForPrice(float *big.Float) (amount0 *big.Int)
	// CalculateSubAmount1ForPrice(float *big.Float) (amount1 *big.Int)
	// Deprecated
	CalculateBuyForSell(amount0In *big.Int) (amount1Out *big.Int)
	CalculateBuyForSellWithOrders(amount0In *big.Int) (amount1Out *big.Int)
	// Deprecated
	CalculateSellForBuy(amount1Out *big.Int) (amount0In *big.Int)
	CalculateSellForBuyWithOrders(amount1Out *big.Int) (amount0In *big.Int)
	CalculateAddLiquidity(amount0 *big.Int, supply *big.Int) (liquidity *big.Int, amount1 *big.Int)
	CheckSwap(amount0In, amount1Out *big.Int) error
	CheckMint(amount0, maxAmount1, totalSupply *big.Int) (err error)
	CheckCreate(amount0, amount1 *big.Int) (err error)
	CheckBurn(liquidity, minAmount0, minAmount1, totalSupply *big.Int) error
}

type RSwap interface {
	// Deprecated
	// ExportV1(state *types.AppState, id types.CoinID, value *big.Int, bipValue *big.Int) *big.Int

	Export(state *types.AppState)
	SwapPool(coin0, coin1 types.CoinID) (reserve0, reserve1 *big.Int, id uint32)
	GetSwapper(coin0, coin1 types.CoinID) EditableChecker
	SwapPoolExist(coin0, coin1 types.CoinID) bool
	PairCalculateBuyForSell(coin0, coin1 types.CoinID, amount0In *big.Int) (amount1Out *big.Int, err error)
	PairCalculateSellForBuy(coin0, coin1 types.CoinID, amount1Out *big.Int) (amount0In *big.Int, err error)
}

type Swap struct {
	muPairs       sync.RWMutex
	pairs         map[pairKey]*Pair
	dirties       map[pairKey]struct{}
	dirtiesOrders map[pairKey]struct{}

	muNextID    sync.Mutex
	nextID      uint32
	dirtyNextID bool

	muNextOrdersID    sync.Mutex
	nextOrderID       uint32
	dirtyNextOrdersID bool

	bus *bus.Bus
	db  atomic.Value
}

func (s *Swap) getOrderedDirtyPairs() []pairKey {
	keys := make([]pairKey, 0, len(s.dirties))
	for k := range s.dirties {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].bytes(), keys[j].bytes()) == 1
	})

	return keys
}

func (s *Swap) getOrderedDirtyOrderPairs() []pairKey {
	// s.muPairs.RLock() // todo: check
	keys := make([]pairKey, 0, len(s.dirtiesOrders))
	for k := range s.dirtiesOrders {
		keys = append(keys, k)
	}
	// s.muPairs.RUnlock()

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].bytes(), keys[j].bytes()) == 1
	})

	return keys
}

func New(bus *bus.Bus, db *iavl.ImmutableTree) *Swap {
	immutableTree := atomic.Value{}
	immutableTree.Store(db)
	return &Swap{pairs: map[pairKey]*Pair{}, bus: bus, db: immutableTree, dirties: map[pairKey]struct{}{}, dirtiesOrders: map[pairKey]struct{}{}}
}

func (s *Swap) immutableTree() *iavl.ImmutableTree {
	return s.db.Load().(*iavl.ImmutableTree)
}

func (s *Swap) Export(state *types.AppState) {
	s.immutableTree().IterateRange([]byte{mainPrefix, pairDataPrefix}, []byte{mainPrefix, pairDataPrefix + 1}, true, func(key []byte, value []byte) bool {
		if len(key) < 10 {
			return false
		}
		coin0 := types.BytesToCoinID(key[2:6])
		coin1 := types.BytesToCoinID(key[6:10])
		_ = s.Pair(coin0, coin1)

		return false
	})

	for key, pair := range s.pairs {
		if pair == nil {
			continue
		}
		var orders []types.Order

		allOrders := pair.loadAllOrders(s.immutableTree())
		for _, limit := range allOrders {
			orders = append(orders, types.Order{
				IsSale:  !limit.isBuy,
				Volume0: limit.WantBuy.String(),
				Volume1: limit.WantSell.String(),
				ID:      uint64(limit.id),
				Owner:   limit.Owner,
			})
		}

		reserve0, reserve1 := pair.Reserves()
		swap := types.Pool{
			Coin0:    uint64(key.Coin0),
			Coin1:    uint64(key.Coin1),
			Reserve0: reserve0.String(),
			Reserve1: reserve1.String(),
			ID:       uint64(pair.GetID()),
			Orders:   orders,
		}

		state.Pools = append(state.Pools, swap)
		state.NextOrderID = uint64(s.loadNextOrdersID())
	}

	sort.Slice(state.Pools, func(i, j int) bool {
		return strconv.Itoa(int(state.Pools[i].Coin0))+"-"+strconv.Itoa(int(state.Pools[i].Coin1)) < strconv.Itoa(int(state.Pools[j].Coin0))+"-"+strconv.Itoa(int(state.Pools[j].Coin1))
	})

}

func (s *Swap) Import(state *types.AppState) {
	for _, pool := range state.Pools {
		coin0 := types.CoinID(pool.Coin0)
		coin1 := types.CoinID(pool.Coin1)
		reserve0 := helpers.StringToBigInt(pool.Reserve0)
		reserve1 := helpers.StringToBigInt(pool.Reserve1)
		pair := s.ReturnPair(coin0, coin1)
		*pair.ID = uint32(pool.ID)
		pair.Reserve0.Set(reserve0)
		pair.Reserve1.Set(reserve1)
		s.bus.Checker().AddCoin(coin0, reserve0)
		s.bus.Checker().AddCoin(coin1, reserve1)
		pair.markDirty()
		s.incID()
		for _, order := range pool.Orders {
			key := pair.pairKey
			if !order.IsSale {
				key = key.reverse()
			}
			s.PairAddOrderWithID(key.Coin0, key.Coin1, helpers.StringToBigInt(order.Volume0), helpers.StringToBigInt(order.Volume1), order.Owner, uint32(order.ID))
		}
		s.nextOrderID = uint32(state.NextOrderID)
		s.dirtyNextID = true
	}
}

const mainPrefix = byte('s')

const pairDataPrefix = 'd'
const pairOrdersPrefix = 'o'
const totalPairIDPrefix = 'i'
const totalOrdersIDPrefix = 'n'

type pairData struct {
	*sync.RWMutex
	Reserve0  *big.Int
	Reserve1  *big.Int
	ID        *uint32
	markDirty func()
}

func (pd *pairData) Reserves() (reserve0 *big.Int, reserve1 *big.Int) {
	pd.RLock()
	defer pd.RUnlock()
	return new(big.Int).Set(pd.Reserve0), new(big.Int).Set(pd.Reserve1)
}

func (pd *pairData) Price() *big.Float {
	pd.RLock()
	defer pd.RUnlock()

	return CalcPriceSell(pd.Reserve0, pd.Reserve1)
}

func (pd *pairData) reverse() *pairData {
	return &pairData{
		RWMutex:   pd.RWMutex,
		Reserve0:  pd.Reserve1,
		Reserve1:  pd.Reserve0,
		ID:        pd.ID,
		markDirty: pd.markDirty,
	}
}

func (s *Swap) CheckSwap(coin0, coin1 types.CoinID, amount0In, amount1Out *big.Int) error {
	return s.Pair(coin0, coin1).checkSwap(amount0In, big.NewInt(0), big.NewInt(0), amount1Out)
}
func (p *Pair) CheckSwap(amount0In, amount1Out *big.Int) error {
	return p.checkSwap(amount0In, big.NewInt(0), big.NewInt(0), amount1Out)
}
func (p *Pair) Exists() bool {
	return p != nil
}
func (p *Pair) AddLastSwapStep(amount0In, amount1Out *big.Int) EditableChecker {
	reserve0, reserve1 := p.Reserves()
	return &Pair{
		pairKey: p.pairKey,
		pairData: &pairData{
			RWMutex:   &sync.RWMutex{},
			Reserve0:  reserve0.Add(reserve0, amount0In),
			Reserve1:  reserve1.Sub(reserve1, amount1Out),
			ID:        p.ID,
			markDirty: func() {},
		},
		sellOrders:          p.sellOrders,
		buyOrders:           p.buyOrders,
		dirtyOrders:         p.dirtyOrders,
		markDirtyOrders:     func() {},
		loadHigherOrders:    p.loadHigherOrders,
		loadLowerOrders:     p.loadLowerOrders,
		getLastTotalOrderID: nil,
	}
}

func (p *Pair) AddLastSwapStepWithOrders(amount0In, amount1Out *big.Int) EditableChecker {
	amount1OutCalc, orders := p.calculateBuyForSellWithOrders(amount0In)
	if amount1OutCalc.Cmp(amount1Out) != 0 {
		log.Println("AddLastSwapStepWithOrders error", amount1OutCalc, amount1Out)
	}
	reserve0, reserve1 := p.Reserves()

	dirties := make(map[uint32]*Limit, len(p.dirtyOrders.orders))
	p.lockOrders.Lock()
	for k, v := range p.dirtyOrders.orders {
		dirties[k] = v
	}
	p.lockOrders.Unlock()

	pair := &Pair{
		pairKey: p.pairKey,
		pairData: &pairData{
			RWMutex:  &sync.RWMutex{},
			Reserve0: reserve0,
			Reserve1: reserve1,
			ID:       p.ID,
			markDirty: func() {
			},
		},
		sellOrders: &limits{
			higher: p.sellOrders.higher,
			lower:  p.sellOrders.lower,
		},
		buyOrders: &limits{
			higher: p.buyOrders.higher,
			lower:  p.buyOrders.lower,
		},
		dirtyOrders: &dirtyOrders{
			orders: dirties,
		},
		markDirtyOrders:     p.markDirtyOrders,
		loadHigherOrders:    p.loadHigherOrders,
		loadLowerOrders:     p.loadLowerOrders,
		getLastTotalOrderID: nil,
	}
	commission0orders, commission1orders, amount0, amount1, _ := CalcDiffPool(amount0In, amount1Out, orders)

	if amount0.Sign() != 0 || amount1.Sign() != 0 {
		pair.update(amount0, big.NewInt(0).Neg(amount1))
	}

	pair.update(commission0orders, commission1orders)

	oo := make([]*Limit, 0, len(orders))
	for _, order := range orders {
		oo = append(oo, &Limit{
			isBuy:        order.isBuy,
			WantBuy:      big.NewInt(0).Set(order.WantBuy),
			WantSell:     big.NewInt(0).Set(order.WantSell),
			Owner:        order.Owner,
			pairKey:      order.pairKey,
			oldSortPrice: big.NewFloat(0).Set(order.oldSortPrice),
			id:           order.id,
			RWMutex:      &sync.RWMutex{},
		})
	}

	pair.updateOrders(oo)

	return pair
}

func (p *Pair) Reverse() EditableChecker {
	return p.reverse()
}
func (p *Pair) reverse() *Pair {
	return &Pair{
		pairKey:             p.pairKey.reverse(),
		pairData:            p.pairData.reverse(),
		buyOrders:           p.buyOrders,
		sellOrders:          p.sellOrders,
		markDirtyOrders:     p.markDirtyOrders,
		dirtyOrders:         p.dirtyOrders,
		loadHigherOrders:    p.loadLowerOrders,
		loadLowerOrders:     p.loadHigherOrders,
		getLastTotalOrderID: p.getLastTotalOrderID,
	}
}

func (p *Pair) Price() *big.Float {
	return p.pairData.Price()
}

func (p *Pair) SortPrice() *big.Float {
	if p.isSorted() {
		return p.pairData.Price()
	}
	return p.pairData.reverse().Price()
}

func (pk pairKey) bytes() []byte {
	key := pk.sort()
	return append(key.Coin0.Bytes(), key.Coin1.Bytes()...)
}

func (pk pairKey) pathData() []byte {
	return append([]byte{pairDataPrefix}, pk.bytes()...)
}
func (pk pairKey) pathOrders() []byte {
	return append([]byte{pairOrdersPrefix}, pk.sort().bytes()...)
}

func pricePath(key pairKey, price *big.Float, id uint32, isSale bool) []byte {
	var pricePath []byte

	text := price.Text('e', 18)
	split := strings.Split(text, "e")
	if len(split) != 2 {
		panic("p")
	}

	{
		// порядок
		bString, err := strconv.Atoi(split[1])
		if err != nil {
			panic(err)
		}
		// log.Println("c p", split[1])
		b := byte(bString + math.MaxInt8)
		pricePath = append(pricePath, b)
	}

	sprintf := fmt.Sprintf("%v", price.Text('f', 18))
	// log.Println("c m", sprintf)
	pricePath = append(pricePath, []byte(sprintf)...)

	byteID := make([]byte, 4)
	binary.BigEndian.PutUint32(byteID, id)

	var saleByte byte = 0
	if isSale {
		saleByte = 1
		// log.Println("c s", saleByte)
	}
	return append(append(append(append([]byte{mainPrefix}, key.pathOrders()...), saleByte), pricePath...), byteID...)
}

func (p *Pair) getDirtyOrdersList() []*Limit {
	dirtiesOrders := make([]*Limit, 0, len(p.dirtyOrders.orders))
	for _, limit := range p.dirtyOrders.orders {
		dirtiesOrders = append(dirtiesOrders, limit)
	}
	sort.SliceStable(dirtiesOrders, func(i, j int) bool {
		return dirtiesOrders[i].id > dirtiesOrders[j].id
	})
	return dirtiesOrders
}

func (s *Swap) Commit(db *iavl.MutableTree, version int64) error {
	basePath := []byte{mainPrefix}

	s.muNextID.Lock()
	if s.dirtyNextID {
		s.dirtyNextID = false
		b, err := rlp.EncodeToBytes(s.nextID)
		if err != nil {
			return err
		}
		db.Set([]byte{mainPrefix, totalPairIDPrefix}, b)
	}
	s.muNextID.Unlock()

	s.muNextOrdersID.Lock()
	if s.dirtyNextOrdersID {
		s.dirtyNextOrdersID = false
		b, err := rlp.EncodeToBytes(s.nextOrderID)
		if err != nil {
			return err
		}
		db.Set([]byte{mainPrefix, totalOrdersIDPrefix}, b)
	}
	s.muNextOrdersID.Unlock()

	s.muPairs.RLock()
	defer s.muPairs.RUnlock()

	for _, key := range s.getOrderedDirtyPairs() {
		pair, _ := s.pair(key)
		pairDataBytes, err := rlp.EncodeToBytes(pair.pairData)
		if err != nil {
			return err
		}
		db.Set(append(basePath, key.pathData()...), pairDataBytes)
	}
	s.dirties = map[pairKey]struct{}{}

	for _, key := range s.getOrderedDirtyOrderPairs() {
		pair, _ := s.pair(key)
		pair.lockOrders.Lock()
		for _, limit := range pair.getDirtyOrdersList() {

			oldPath := pricePath(key, limit.OldSortPrice(), limit.id, !limit.isBuy)

			if limit.isEmpty() {
				if limit.WantBuy.Sign() != 0 || limit.WantSell.Sign() != 0 {
					panic(fmt.Sprintf("order %d has one zero volume: %s, %s. Sell %v", limit.id, limit.WantBuy, limit.WantSell, !limit.isBuy))
				}
				db.Remove(oldPath)
				// log.Printf("remove old path %q, %s, %s. Sell %v", oldPath, limit.WantBuy, limit.WantSell, !limit.isBuy)
				continue
			}

			newPath := pricePath(key, limit.ReCalcOldSortPrice(), limit.id, !limit.isBuy)

			pairOrderBytes, err := rlp.EncodeToBytes(limit)
			if err != nil {
				return err
			}

			db.Set(newPath, pairOrderBytes)
			// log.Printf("new path %q, %s, %s. Sell %v", newPath, limit.WantBuy, limit.WantSell, !limit.isBuy)
			if !bytes.Equal(oldPath, newPath) && db.Has(oldPath) {
				db.Remove(oldPath) // the price can change minimally due to rounding off the ratio of the remaining volumes
				// log.Printf("remove old path %q, %s, %s. Sell %v", oldPath, limit.WantBuy, limit.WantSell, !limit.isBuy)
			}
		}
		pair.lockOrders.Unlock()
		pair.dirtyOrders.orders = make(map[uint32]*Limit)
	}
	s.dirtiesOrders = map[pairKey]struct{}{}
	return nil
}

func (s *Swap) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	s.db.Store(immutableTree)
}

func (s *Swap) SwapPoolExist(coin0, coin1 types.CoinID) bool {
	return s.Pair(coin0, coin1) != nil
}

func (s *Swap) pair(key pairKey) (*Pair, bool) {
	pair, ok := s.pairs[key.sort()]
	if pair == nil {
		return nil, ok
	}
	if key.isSorted() {
		return pair, true
	}
	return pair.reverse(), true
}

func (s *Swap) SwapPool(coinA, coinB types.CoinID) (reserve0, reserve1 *big.Int, id uint32) {
	pair := s.Pair(coinA, coinB)
	if pair == nil {
		return nil, nil, 0
	}
	reserve0, reserve1 = pair.Reserves()
	return reserve0, reserve1, *pair.ID
}

func (s *Swap) GetSwapper(coinA, coinB types.CoinID) EditableChecker {
	return s.Pair(coinA, coinB)
}

func (s *Swap) Pair(coin0, coin1 types.CoinID) *Pair {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{Coin0: coin0, Coin1: coin1}
	pair, ok := s.pair(key)
	if ok {
		return pair
	}

	pathPair := append([]byte{mainPrefix}, key.sort().pathData()...)
	_, data := s.immutableTree().Get(pathPair)
	if len(data) == 0 {
		s.pairs[key.sort()] = nil
		return nil
	}

	pair = s.addPair(key)
	err := rlp.DecodeBytes(data, pair.pairData)
	if err != nil {
		panic(err)
	}

	if !key.isSorted() {
		return pair.reverse()
	}

	return pair
}

func (s *Swap) PairCalculateSellForBuy(coin0, coin1 types.CoinID, amount1Out *big.Int) (amount0In *big.Int, err error) {
	pair := s.Pair(coin0, coin1)
	if pair == nil {
		return nil, ErrorNotExist
	}
	value := pair.CalculateSellForBuy(amount1Out)
	if value == nil {
		return nil, ErrorInsufficientLiquidity
	}
	return value, nil
}

func (s *Swap) PairCalculateBuyForSell(coin0, coin1 types.CoinID, amount0In *big.Int) (amount1Out *big.Int, err error) {
	pair := s.Pair(coin0, coin1)
	if pair == nil {
		return nil, ErrorNotExist
	}
	value := pair.CalculateBuyForSell(amount0In)
	if value == nil {
		return nil, ErrorInsufficientLiquidity
	}
	return value, nil
}

func (s *Swap) PairMint(coin0, coin1 types.CoinID, amount0, maxAmount1, totalSupply *big.Int) (*big.Int, *big.Int, *big.Int) {
	pair := s.Pair(coin0, coin1)
	oldReserve0, oldReserve1 := pair.Reserves()
	liquidity := pair.Mint(amount0, maxAmount1, totalSupply)
	newReserve0, newReserve1 := pair.Reserves()

	balance0 := new(big.Int).Sub(newReserve0, oldReserve0)
	balance1 := new(big.Int).Sub(newReserve1, oldReserve1)

	s.bus.Checker().AddCoin(coin0, balance0)
	s.bus.Checker().AddCoin(coin1, balance1)

	return balance0, balance1, liquidity
}

func (s *Swap) PairCreate(coin0, coin1 types.CoinID, amount0, amount1 *big.Int) (*big.Int, *big.Int, *big.Int, uint32) {
	pair := s.ReturnPair(coin0, coin1)
	id := s.incID()
	*pair.ID = id
	oldReserve0, oldReserve1 := pair.Reserves()
	liquidity := pair.Create(amount0, amount1)
	newReserve0, newReserve1 := pair.Reserves()

	balance0 := new(big.Int).Sub(newReserve0, oldReserve0)
	balance1 := new(big.Int).Sub(newReserve1, oldReserve1)

	s.bus.Checker().AddCoin(coin0, balance0)
	s.bus.Checker().AddCoin(coin1, balance1)

	return balance0, balance1, liquidity, id
}

func (s *Swap) PairBurn(coin0, coin1 types.CoinID, liquidity, minAmount0, minAmount1, totalSupply *big.Int) (*big.Int, *big.Int) {
	pair := s.Pair(coin0, coin1)
	oldReserve0, oldReserve1 := pair.Reserves()
	_, _ = pair.Burn(liquidity, minAmount0, minAmount1, totalSupply)
	newReserve0, newReserve1 := pair.Reserves()

	balance0 := new(big.Int).Sub(oldReserve0, newReserve0)
	balance1 := new(big.Int).Sub(oldReserve1, newReserve1)

	s.bus.Checker().AddCoin(coin0, new(big.Int).Neg(balance0))
	s.bus.Checker().AddCoin(coin1, new(big.Int).Neg(balance1))

	return balance0, balance1
}

// Deprecated
func (s *Swap) PairSell(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32) {
	pair := s.Pair(coin0, coin1)
	calculatedAmount1Out := pair.CalculateBuyForSell(amount0In)
	if calculatedAmount1Out.Cmp(minAmount1Out) == -1 {
		panic(fmt.Sprintf("calculatedAmount1Out %s less minAmount1Out %s", calculatedAmount1Out, minAmount1Out))
	}
	balance0, balance1 := pair.Swap(amount0In, big.NewInt(0), big.NewInt(0), calculatedAmount1Out)
	s.bus.Checker().AddCoin(coin0, balance0)
	s.bus.Checker().AddCoin(coin1, balance1)
	return balance0, new(big.Int).Neg(balance1), *pair.ID
}

// Deprecated
func (s *Swap) PairBuy(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32) {
	pair := s.Pair(coin0, coin1)
	calculatedAmount0In := pair.CalculateSellForBuy(amount1Out)
	if calculatedAmount0In.Cmp(maxAmount0In) == 1 {
		panic(fmt.Sprintf("calculatedAmount0In %s more maxAmount0In %s", calculatedAmount0In, maxAmount0In))
	}
	balance0, balance1 := pair.Swap(calculatedAmount0In, big.NewInt(0), big.NewInt(0), amount1Out)
	s.bus.Checker().AddCoin(coin0, balance0)
	s.bus.Checker().AddCoin(coin1, balance1)
	return balance0, new(big.Int).Neg(balance1), *pair.ID
}

type pairKey struct {
	Coin0, Coin1 types.CoinID
}

func (pk pairKey) sort() pairKey {
	if pk.isSorted() {
		return pk
	}
	return pk.reverse()
}

func (pk *pairKey) isSorted() bool {
	return pk.Coin0 < pk.Coin1
}

func (pk *pairKey) reverse() pairKey {
	return pairKey{Coin0: pk.Coin1, Coin1: pk.Coin0}
}

var (
	ErrorIdenticalAddresses = errors.New("IDENTICAL_ADDRESSES")
)

func (s *Swap) ReturnPair(coin0, coin1 types.CoinID) *Pair {
	if coin0 == coin1 {
		panic(ErrorIdenticalAddresses)
	}

	pair := s.Pair(coin0, coin1)
	if pair != nil {
		return pair
	}

	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{coin0, coin1}
	pair = s.addPair(key)

	if !key.isSorted() {
		return pair.reverse()
	}

	return pair
}

func (s *Swap) markDirty(key pairKey) func() {
	return func() {
		s.muPairs.Lock()
		defer s.muPairs.Unlock()
		s.dirties[key] = struct{}{}
	}
}
func (s *Swap) markDirtyOrders(key pairKey) func() {
	return func() {
		s.muPairs.Lock()
		defer s.muPairs.Unlock()
		s.dirtiesOrders[key] = struct{}{}
	}
}

func (s *Swap) addPair(key pairKey) *Pair {
	if !key.isSorted() {
		key = key.reverse()
	}
	pair := &Pair{
		pairKey: key,
		pairData: &pairData{
			RWMutex:   &sync.RWMutex{},
			Reserve0:  big.NewInt(0),
			Reserve1:  big.NewInt(0),
			ID:        new(uint32),
			markDirty: s.markDirty(key),
		},
		sellOrders:          &limits{},
		buyOrders:           &limits{},
		dirtyOrders:         &dirtyOrders{orders: make(map[uint32]*Limit)},
		markDirtyOrders:     s.markDirtyOrders(key),
		loadHigherOrders:    s.loadBuyHigherOrders,
		loadLowerOrders:     s.loadSellLowerOrders,
		getLastTotalOrderID: s.incOrdersID,
	}

	s.pairs[key] = pair

	return pair
}

func (s *Swap) incID() uint32 {
	s.muNextID.Lock()
	defer s.muNextID.Unlock()

	id := s.loadNextID()
	s.nextID = id + 1
	s.dirtyNextID = true
	return id
}

func (s *Swap) loadNextID() uint32 {
	if s.nextID != 0 {
		return s.nextID
	}
	_, value := s.immutableTree().Get([]byte{mainPrefix, totalPairIDPrefix})
	if len(value) == 0 {
		return 1
	}
	var id uint32
	if err := rlp.DecodeBytes(value, &id); err != nil {
		panic(err)
	}
	return id
}

func (s *Swap) incOrdersID() uint32 {
	s.muNextOrdersID.Lock()
	defer s.muNextOrdersID.Unlock()

	id := s.loadNextOrdersID()
	s.nextOrderID = id + 1
	s.dirtyNextOrdersID = true
	return id
}

func (s *Swap) loadNextOrdersID() uint32 {
	if s.nextOrderID != 0 {
		return s.nextOrderID
	}
	_, value := s.immutableTree().Get([]byte{mainPrefix, totalOrdersIDPrefix})
	if len(value) == 0 {
		return 1
	}
	var id uint32
	if err := rlp.DecodeBytes(value, &id); err != nil {
		panic(err)
	}
	return id
}

var (
	ErrorInsufficientLiquidityMinted = errors.New("INSUFFICIENT_LIQUIDITY_MINTED")
)

type Balance struct {
	Liquidity *big.Int
	isDirty   bool
}

type Pair struct {
	lockOrders sync.RWMutex
	pairKey
	*pairData
	sellOrders          *limits
	buyOrders           *limits
	dirtyOrders         *dirtyOrders
	markDirtyOrders     func()
	loadHigherOrders    func(pair *Pair, slice []*Limit, limit int) []*Limit
	loadLowerOrders     func(pair *Pair, slice []*Limit, limit int) []*Limit
	getLastTotalOrderID func() uint32
}

func (p *Pair) GetID() uint32 {
	if p == nil {
		return 0
	}
	return *p.ID
}

func (p *Pair) CalculateAddLiquidity(amount0 *big.Int, totalSupply *big.Int) (liquidity *big.Int, amount1 *big.Int) {
	reserve0, reserve1 := p.Reserves()
	return new(big.Int).Div(new(big.Int).Mul(totalSupply, amount0), reserve0), new(big.Int).Div(new(big.Int).Mul(amount0, reserve1), reserve0)
}

func (p *Pair) Mint(amount0, amount1, totalSupply *big.Int) (liquidity *big.Int) {
	liquidity, amount1 = p.CalculateAddLiquidity(amount0, totalSupply)
	if liquidity.Sign() != 1 {
		panic(ErrorInsufficientLiquidityMinted)
	}
	p.update(amount0, amount1)
	return new(big.Int).Set(liquidity)
}

func (p *Pair) Create(amount0, amount1 *big.Int) (liquidity *big.Int) {
	liquidity = startingSupply(amount0, amount1)

	if liquidity.Cmp(Bound) != 1 {
		panic(ErrorInsufficientLiquidityMinted)
	}
	p.update(amount0, amount1)
	return new(big.Int).Set(liquidity)
}

func (p *Pair) CheckMint(amount0, maxAmount1, totalSupply *big.Int) (err error) {

	liquidity, amount1 := p.CalculateAddLiquidity(amount0, totalSupply)
	if amount1.Cmp(maxAmount1) == 1 {
		return ErrorInsufficientInputAmount
	}

	if liquidity.Sign() != 1 {
		return ErrorInsufficientLiquidityMinted
	}

	return nil
}
func (p *Pair) CheckCreate(amount0, maxAmount1 *big.Int) (err error) {
	liquidity := startingSupply(amount0, maxAmount1)

	if liquidity.Cmp(Bound) != 1 {
		return ErrorInsufficientLiquidityMinted
	}

	return nil
}

var (
	ErrorInsufficientLiquidityBurned  = errors.New("INSUFFICIENT_LIQUIDITY_BURNED")
	ErrorInsufficientLiquidityBalance = errors.New("INSUFFICIENT_LIQUIDITY_BALANCE")
	ErrorNotExist                     = errors.New("PAIR_NOT_EXISTS")
)

func (p *Pair) Burn(liquidity, minAmount0, minAmount1, totalSupply *big.Int) (amount0, amount1 *big.Int) {
	amount0, amount1 = p.Amounts(liquidity, totalSupply)

	if amount0.Cmp(minAmount0) == -1 || amount1.Cmp(minAmount1) == -1 {
		panic(ErrorInsufficientLiquidityBurned)
	}

	p.update(new(big.Int).Neg(amount0), new(big.Int).Neg(amount1))

	return amount0, amount1
}

func (p *Pair) CheckBurn(liquidity, minAmount0, minAmount1, totalSupply *big.Int) error {
	if p == nil {
		return ErrorNotExist
	}
	amount0, amount1 := p.Amounts(liquidity, totalSupply)

	if amount0.Cmp(minAmount0) == -1 || amount1.Cmp(minAmount1) == -1 {
		return ErrorInsufficientLiquidityBurned
	}

	return nil
}

var (
	ErrorK                        = errors.New("K")
	ErrorInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrorInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrorInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
)

// Deprecated
func (p *Pair) CalculateBuyForSellAllowNeg(amount0In *big.Int) (amount1Out *big.Int) {
	if amount0In.Sign() == -1 {
		amount1Out := p.reverse().CalculateSellForBuy(big.NewInt(0).Neg(amount0In))
		return amount1Out.Neg(amount1Out)
	}

	reserve0, reserve1 := p.Reserves()
	kAdjusted := new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))
	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0In, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(commission)))
	amount1Out = new(big.Int).Sub(reserve1, new(big.Int).Quo(kAdjusted, new(big.Int).Mul(balance0Adjusted, big.NewInt(1000))))
	amount1Out = new(big.Int).Sub(amount1Out, big.NewInt(1))
	if amount1Out.Sign() != 1 {
		return nil
	}

	return amount1Out
}

// reserve1-(reserve0*reserve1)/((amount0+reserve0)-amount0*0.002)
func (p *Pair) CalculateBuyForSell(amount0In *big.Int) (amount1Out *big.Int) {
	reserve0, reserve1 := p.Reserves()
	kAdjusted := new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))
	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0In, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(commission)))
	amount1Out = new(big.Int).Sub(reserve1, new(big.Int).Quo(kAdjusted, new(big.Int).Mul(balance0Adjusted, big.NewInt(1000))))
	amount1Out = new(big.Int).Sub(amount1Out, big.NewInt(1))
	if amount1Out.Sign() != 1 {
		return nil
	}

	return amount1Out
}

// Deprecated
func (p *Pair) CalculateSellForBuyAllowNeg(amount1Out *big.Int) (amount0In *big.Int) {
	if amount1Out.Sign() == -1 {
		amount0In := p.reverse().CalculateBuyForSell(big.NewInt(0).Neg(amount1Out))
		return amount0In.Neg(amount0In)
	}

	reserve0, reserve1 := p.Reserves()
	k := new(big.Int).Mul(reserve0, reserve1)
	if amount1Out.Cmp(reserve1) != -1 {
		return nil
	}
	kAdjusted := new(big.Int).Mul(k, big.NewInt(1000000))
	balance1Adjusted := new(big.Int).Mul(new(big.Int).Add(new(big.Int).Neg(amount1Out), reserve1), big.NewInt(1000))
	amount0In = new(big.Int).Quo(new(big.Int).Sub(new(big.Int).Quo(kAdjusted, balance1Adjusted), new(big.Int).Mul(reserve0, big.NewInt(1000))), big.NewInt(1000-commission))
	return new(big.Int).Add(amount0In, big.NewInt(1))
}

// (reserve0*reserve1/(reserve1-amount1)-reserve0)/0.998
func (p *Pair) CalculateSellForBuy(amount1Out *big.Int) (amount0In *big.Int) {
	reserve0, reserve1 := p.Reserves()
	if amount1Out.Cmp(reserve1) == 1 {
		return nil
	}
	k := new(big.Int).Mul(reserve0, reserve1)
	if amount1Out.Cmp(reserve1) != -1 {
		return nil
	}
	kAdjusted := new(big.Int).Mul(k, big.NewInt(1000000))
	balance1Adjusted := new(big.Int).Mul(new(big.Int).Add(new(big.Int).Neg(amount1Out), reserve1), big.NewInt(1000))
	amount0In = new(big.Int).Quo(new(big.Int).Sub(new(big.Int).Quo(kAdjusted, balance1Adjusted), new(big.Int).Mul(reserve0, big.NewInt(1000))), big.NewInt(1000-commission))
	return new(big.Int).Add(amount0In, big.NewInt(1))
}

// Deprecated
func (p *Pair) Swap(amount0In, amount1In, amount0Out, amount1Out *big.Int) (amount0, amount1 *big.Int) {
	if amount0Out.Sign() != 1 && amount1Out.Sign() != 1 {
		panic(ErrorInsufficientOutputAmount)
	}

	reserve0, reserve1 := p.Reserves()

	if amount0Out.Cmp(reserve0) == 1 || amount1Out.Cmp(reserve1) == 1 {
		panic(ErrorInsufficientLiquidity)
	}

	amount0 = new(big.Int).Sub(amount0In, amount0Out)
	amount1 = new(big.Int).Sub(amount1In, amount1Out)

	if amount0.Sign() != 1 && amount1.Sign() != 1 {
		panic(ErrorInsufficientInputAmount)
	}

	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(commission)))
	balance1Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount1, reserve1), big.NewInt(1000)), new(big.Int).Mul(amount1In, big.NewInt(commission)))

	if new(big.Int).Mul(balance0Adjusted, balance1Adjusted).Cmp(new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))) == -1 {
		panic(ErrorK)
	}

	p.update(amount0, amount1)

	return amount0, amount1
}

func (p *Pair) checkSwap(amount0In, amount1In, amount0Out, amount1Out *big.Int) (err error) {
	reserve0, reserve1 := p.Reserves()
	if amount0Out.Cmp(reserve0) == 1 || amount1Out.Cmp(reserve1) == 1 {
		return ErrorInsufficientLiquidity
	}

	if amount0Out.Sign() != 1 && amount1Out.Sign() != 1 {
		return ErrorInsufficientOutputAmount
	}

	amount0 := new(big.Int).Sub(amount0In, amount0Out)
	amount1 := new(big.Int).Sub(amount1In, amount1Out)

	if amount0.Sign() != 1 && amount1.Sign() != 1 {
		return ErrorInsufficientInputAmount
	}

	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(commission)))
	balance1Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount1, reserve1), big.NewInt(1000)), new(big.Int).Mul(amount1In, big.NewInt(commission)))

	if new(big.Int).Mul(balance0Adjusted, balance1Adjusted).Cmp(new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))) == -1 {
		return ErrorK
	}
	return nil
}

func (p *Pair) update(amount0, amount1 *big.Int) {
	p.markDirty()

	p.pairData.Lock()
	defer p.pairData.Unlock()

	p.Reserve0.Add(p.Reserve0, amount0)
	p.Reserve1.Add(p.Reserve1, amount1)
}

func (p *Pair) Amounts(liquidity, totalSupply *big.Int) (amount0 *big.Int, amount1 *big.Int) {
	p.pairData.RLock()
	defer p.pairData.RUnlock()
	amount0 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.Reserve0), totalSupply)
	amount1 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.Reserve1), totalSupply)
	return amount0, amount1
}

func startingSupply(amount0 *big.Int, amount1 *big.Int) *big.Int {
	mul := new(big.Int).Mul(amount0, amount1)
	return new(big.Int).Sqrt(mul)
}
