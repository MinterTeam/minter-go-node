package swap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/events"
	"math"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
)

type SwapV2 struct {
	muPairs       sync.RWMutex
	pairs         map[PairKey]*PairV2
	dirties       map[PairKey]struct{}
	dirtiesOrders map[PairKey]struct{}

	muNextID    sync.Mutex
	nextID      uint32
	dirtyNextID bool

	muNextOrdersID    sync.Mutex
	nextOrderID       uint32
	dirtyNextOrdersID bool

	version int

	bus *bus.Bus
	db  atomic.Value
}

func (s *SwapV2) ExpireOrders(beforeHeight uint64) {
	var orders []*Limit
	s.immutableTree().IterateRange(pathOrder(0), pathOrder(math.MaxUint32), true, func(key []byte, value []byte) bool {
		if value == nil {
			return false
		}

		id := binary.BigEndian.Uint32(key[1:])

		order := &Limit{
			id:           id,
			oldSortPrice: new(big.Float).SetPrec(Precision),
			mu:           new(sync.RWMutex),
		}
		err := rlp.DecodeBytes(value, order)
		if err != nil {
			panic(err)
		}

		if order.Height > beforeHeight {
			return true
		}

		orders = append(orders, order)

		return false
	})

	for _, order := range orders {
		//fmt.Println(order)
		coin, volume := s.removeLimitOrder(order)
		if volume.Sign() == 0 {
			continue
		}
		s.bus.Accounts().AddBalance(order.Owner, coin, volume)
		s.bus.Events().AddEvent(&events.OrderExpiredEvent{
			ID:      uint64(order.ID()),
			Address: order.Owner,
			Coin:    uint64(coin),
			Amount:  volume.String(),
		})
	}
}

func (s *SwapV2) getOrderedDirtyPairs() []PairKey {
	keys := make([]PairKey, 0, len(s.dirties))
	for k := range s.dirties {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].bytes(), keys[j].bytes()) == 1
	})

	return keys
}

func (s *SwapV2) getOrderedDirtyOrderPairs() []PairKey {
	keys := make([]PairKey, 0, len(s.dirtiesOrders))
	for k := range s.dirtiesOrders {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].bytes(), keys[j].bytes()) == 1
	})

	return keys
}

func NewV2(bus *bus.Bus, db *iavl.ImmutableTree) *SwapV2 {
	immutableTree := atomic.Value{}
	immutableTree.Store(db)
	return &SwapV2{pairs: map[PairKey]*PairV2{}, bus: bus, db: immutableTree, dirties: map[PairKey]struct{}{}, dirtiesOrders: map[PairKey]struct{}{}}
}

func (s *SwapV2) immutableTree() *iavl.ImmutableTree {
	return s.db.Load().(*iavl.ImmutableTree)
}

func (s *SwapV2) Export(state *types.AppState) {
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
				IsSale:  !limit.IsBuy,
				Volume0: limit.WantBuy.String(),
				Volume1: limit.WantSell.String(),
				ID:      uint64(limit.id),
				Owner:   limit.Owner,
				Height:  limit.Height,
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

func (s *SwapV2) Import(state *types.AppState) {
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
			v0 := helpers.StringToBigInt(order.Volume0)
			v1 := helpers.StringToBigInt(order.Volume1)
			pair0 := pair
			if !order.IsSale {
				pair0 = pair.reverse()
				v0, v1 = v1, v0
			}

			pair0.addOrderWithID(v0, v1, order.Owner, uint32(order.ID), order.Height)
			s.bus.Checker().AddCoin(pair0.Coin1, v1)
		}
	}
	if state.NextOrderID > 1 {
		s.nextOrderID = uint32(state.NextOrderID)
		s.dirtyNextOrdersID = true
	}
}

func (s *SwapV2) CheckSwap(coin0, coin1 types.CoinID, amount0In, amount1Out *big.Int) error {
	return s.Pair(coin0, coin1).checkSwap(amount0In, big.NewInt(0), big.NewInt(0), amount1Out)
}
func (p *PairV2) CheckSwap(amount0In, amount1Out *big.Int) error {
	return p.checkSwap(amount0In, big.NewInt(0), big.NewInt(0), amount1Out)
}
func (p *PairV2) Exists() bool {
	return p != nil
}
func (p *PairV2) AddLastSwapStep(amount0In, amount1Out *big.Int) EditableChecker {
	reserve0, reserve1 := p.Reserves()
	return &PairV2{
		lockOrders: &sync.Mutex{},
		PairKey:    p.PairKey,
		pairData: &pairData{
			RWMutex:   &sync.RWMutex{},
			Reserve0:  reserve0.Add(reserve0, amount0In),
			Reserve1:  reserve1.Sub(reserve1, amount1Out),
			ID:        p.ID,
			markDirty: func() {},
		},
		sellOrders:              p.sellOrders,
		buyOrders:               p.buyOrders,
		orders:                  p.orders,
		dirtyOrders:             p.dirtyOrders,
		deletedSellOrders:       p.deletedSellOrders,
		deletedBuyOrders:        p.deletedBuyOrders,
		markDirtyOrders:         func() {},
		loadBuyOrders:           p.loadBuyOrders,
		loadSellOrders:          p.loadSellOrders,
		loadedSellOrders:        p.loadedSellOrders,
		loadedBuyOrders:         p.loadedBuyOrders,
		unsortedDirtyBuyOrders:  p.unsortedDirtyBuyOrders,
		unsortedDirtySellOrders: p.unsortedDirtySellOrders,
		getLastTotalOrderID:     nil,
		loadOrder:               p.loadOrder,
	}
}

func (p *PairV2) Reverse() EditableChecker {
	return p.reverse()
}
func (p *PairV2) IsSorted() bool {
	return p.isSorted()
}
func (p *PairV2) reverse() *PairV2 {
	return &PairV2{
		lockOrders:              p.lockOrders,
		PairKey:                 p.PairKey.reverse(),
		pairData:                p.pairData.reverse(),
		sellOrders:              p.sellOrders,
		buyOrders:               p.buyOrders,
		orders:                  p.orders,
		dirtyOrders:             p.dirtyOrders,
		deletedSellOrders:       p.deletedSellOrders,
		deletedBuyOrders:        p.deletedBuyOrders,
		markDirtyOrders:         p.markDirtyOrders,
		loadBuyOrders:           p.loadSellOrders,
		loadSellOrders:          p.loadBuyOrders,
		loadedSellOrders:        p.loadedSellOrders,
		loadedBuyOrders:         p.loadedBuyOrders,
		unsortedDirtyBuyOrders:  p.unsortedDirtyBuyOrders,
		unsortedDirtySellOrders: p.unsortedDirtySellOrders,
		getLastTotalOrderID:     p.getLastTotalOrderID,
		loadOrder:               p.loadOrder,
	}
}

func (s *SwapV2) Commit(db *iavl.MutableTree, version int64) error {
	basePath := []byte{mainPrefix}

	s.muNextID.Lock()
	if s.dirtyNextID {
		s.dirtyNextID = false
		b, err := rlp.EncodeToBytes(s.nextID)
		if err != nil {
			s.muNextID.Unlock()
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
			s.muNextOrdersID.Unlock()
			panic(err)
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
			panic(err)
			return err
		}
		db.Set(append(basePath, key.pathData()...), pairDataBytes)
	}
	s.dirties = map[PairKey]struct{}{}

	for _, key := range s.getOrderedDirtyOrderPairs() {
		pair, _ := s.pair(key)
		pair.lockOrders.Lock()

		for _, id := range pair.getDirtyOrdersList() {
			limit := pair.getOrder(id)

			if limit.isEmpty() {
				pair.orders.mu.Lock()
				pair.orders.list[limit.id] = nil
				pair.orders.mu.Unlock()
			}

			pathOrderID := pathOrder(limit.id)

			oldSortPrice := limit.OldSortPrice()
			newPath := pricePath(key, limit.reCalcOldSortPrice(), limit.id, !limit.IsBuy)
			if oldSortPrice.Sign() != 0 {
				oldPathOrderList := pricePath(key, oldSortPrice, limit.id, !limit.IsBuy)

				if limit.isEmpty() {
					db.Remove(pathOrderID)
					db.Remove(oldPathOrderList)
					continue
				}

				if !bytes.Equal(oldPathOrderList, newPath) {
					db.Remove(oldPathOrderList)
				}
			}

			if limit.isEmpty() {
				continue
			}

			db.Set(newPath, []byte{})

			pairOrderBytes, err := rlp.EncodeToBytes(limit)
			if err != nil {
				panic(err)
				return err
			}

			db.Set(pathOrderID, pairOrderBytes)
		}

		//lenB := len(pair.buyOrders.ids)
		//pair.loadedBuyOrders.ids = pair.buyOrders.ids[:lenB:lenB]
		//if lenB > 10 {
		//	pair.buyOrders.ids = pair.buyOrders.ids[:10:10]
		//}
		pair.loadedBuyOrders.ids = nil
		pair.buyOrders.ids = nil

		//lenS := len(pair.sellOrders.ids)
		//pair.loadedSellOrders.ids = pair.sellOrders.ids[:lenS:lenS]
		//if lenS > 10 {
		//	pair.sellOrders.ids = pair.sellOrders.ids[:10:10]
		//}
		pair.loadedSellOrders.ids = nil
		pair.sellOrders.ids = nil

		pair.dirtyOrders.mu.Lock()
		pair.dirtyOrders.list = make(map[uint32]struct{})
		pair.dirtyOrders.mu.Unlock()

		pair.deletedBuyOrders.mu.Lock()
		pair.deletedBuyOrders.list = make(map[uint32]struct{})
		pair.deletedBuyOrders.mu.Unlock()

		pair.deletedSellOrders.mu.Lock()
		pair.deletedSellOrders.list = make(map[uint32]struct{})
		pair.deletedSellOrders.mu.Unlock()

		pair.unsortedDirtyBuyOrders.mu.Lock()
		pair.unsortedDirtyBuyOrders.list = make(map[uint32]struct{})
		pair.unsortedDirtyBuyOrders.mu.Unlock()

		pair.unsortedDirtySellOrders.mu.Lock()
		pair.unsortedDirtySellOrders.list = make(map[uint32]struct{})
		pair.unsortedDirtySellOrders.mu.Unlock()

		pair.lockOrders.Unlock()
	}
	s.dirtiesOrders = map[PairKey]struct{}{}
	return nil
}

func (s *SwapV2) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	s.db.Store(immutableTree)
}

func (s *SwapV2) SwapPoolExist(coin0, coin1 types.CoinID) bool {
	return s.Pair(coin0, coin1) != nil
}

func (s *SwapV2) pair(key PairKey) (*PairV2, bool) {
	pair, ok := s.pairs[key.sort()]
	if pair == nil {
		return nil, ok
	}
	if key.isSorted() {
		return pair, true
	}
	return pair.reverse(), true
}

func (s *SwapV2) SwapPool(coinA, coinB types.CoinID) (reserve0, reserve1 *big.Int, id uint32) {
	pair := s.Pair(coinA, coinB)
	if pair == nil {
		return nil, nil, 0
	}
	reserve0, reserve1 = pair.Reserves()
	return reserve0, reserve1, *pair.ID
}

func (s *SwapV2) GetSwapper(coinA, coinB types.CoinID) EditableChecker {
	return s.Pair(coinA, coinB)
}

func (s *SwapV2) Pair(coin0, coin1 types.CoinID) *PairV2 {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := PairKey{Coin0: coin0, Coin1: coin1}
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

// Deprecated
func (s *SwapV2) PairCalculateSellForBuy(coin0, coin1 types.CoinID, amount1Out *big.Int) (amount0In *big.Int, err error) {
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

// Deprecated
func (s *SwapV2) PairCalculateBuyForSell(coin0, coin1 types.CoinID, amount0In *big.Int) (amount1Out *big.Int, err error) {
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

func (s *SwapV2) PairMint(coin0, coin1 types.CoinID, amount0, maxAmount1, totalSupply *big.Int) (*big.Int, *big.Int, *big.Int) {
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

func (s *SwapV2) PairCreate(coin0, coin1 types.CoinID, amount0, amount1 *big.Int) (*big.Int, *big.Int, *big.Int, uint32) {
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

func (s *SwapV2) PairBurn(coin0, coin1 types.CoinID, liquidity, minAmount0, minAmount1, totalSupply *big.Int) (*big.Int, *big.Int) {
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
func (s *SwapV2) PairSell(coin0, coin1 types.CoinID, amount0In, minAmount1Out *big.Int) (*big.Int, *big.Int, uint32) {
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
func (s *SwapV2) PairBuy(coin0, coin1 types.CoinID, maxAmount0In, amount1Out *big.Int) (*big.Int, *big.Int, uint32) {
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

func (s *SwapV2) ReturnPair(coin0, coin1 types.CoinID) *PairV2 {
	if coin0 == coin1 {
		panic(ErrorIdenticalAddresses)
	}

	pair := s.Pair(coin0, coin1)
	if pair != nil {
		return pair
	}

	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := PairKey{coin0, coin1}
	pair = s.addPair(key)

	if !key.isSorted() {
		return pair.reverse()
	}

	return pair
}

func (s *SwapV2) markDirty(key PairKey) func() {
	return func() {
		s.muPairs.Lock()
		defer s.muPairs.Unlock()
		s.dirties[key] = struct{}{}
	}
}
func (s *SwapV2) markDirtyOrders(key PairKey) func() {
	return func() {
		s.muPairs.Lock()
		defer s.muPairs.Unlock()
		s.dirtiesOrders[key] = struct{}{}
	}
}

func (s *SwapV2) addPair(key PairKey) *PairV2 {
	if !key.isSorted() {
		key = key.reverse()
	}
	pair := &PairV2{
		lockOrders: &sync.Mutex{},
		PairKey:    key,
		pairData: &pairData{
			RWMutex:   &sync.RWMutex{},
			Reserve0:  big.NewInt(0),
			Reserve1:  big.NewInt(0),
			ID:        new(uint32),
			markDirty: s.markDirty(key),
		},
		sellOrders:              &limits{},
		buyOrders:               &limits{},
		orders:                  &orderList{list: make(map[uint32]*Limit), mu: sync.RWMutex{}},
		dirtyOrders:             &orderDirties{list: make(map[uint32]struct{}), mu: sync.RWMutex{}},
		deletedSellOrders:       &orderDirties{list: make(map[uint32]struct{}), mu: sync.RWMutex{}},
		deletedBuyOrders:        &orderDirties{list: make(map[uint32]struct{}), mu: sync.RWMutex{}},
		markDirtyOrders:         s.markDirtyOrders(key),
		loadBuyOrders:           s.loadBuyOrders,
		loadSellOrders:          s.loadSellOrders,
		loadedSellOrders:        &limits{},
		loadedBuyOrders:         &limits{},
		unsortedDirtyBuyOrders:  &orderDirties{list: make(map[uint32]struct{}), mu: sync.RWMutex{}},
		unsortedDirtySellOrders: &orderDirties{list: make(map[uint32]struct{}), mu: sync.RWMutex{}},
		getLastTotalOrderID:     s.incOrdersID,
		loadOrder:               s.loadOrder,
	}

	s.pairs[key] = pair

	return pair
}

func (s *SwapV2) incID() uint32 {
	s.muNextID.Lock()
	defer s.muNextID.Unlock()

	id := s.loadNextID()
	s.nextID = id + 1
	s.dirtyNextID = true
	return id
}

func (s *SwapV2) loadNextID() uint32 {
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

func (s *SwapV2) incOrdersID() uint32 {
	s.muNextOrdersID.Lock()
	defer s.muNextOrdersID.Unlock()

	id := s.loadNextOrdersID()
	s.nextOrderID = id + 1
	s.dirtyNextOrdersID = true
	return id
}

func (s *SwapV2) loadNextOrdersID() uint32 {
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

type PairV2 struct {
	lockOrders *sync.Mutex
	PairKey
	*pairData
	sellOrders              *limits
	buyOrders               *limits
	orders                  *orderList
	dirtyOrders             *orderDirties
	deletedSellOrders       *orderDirties
	deletedBuyOrders        *orderDirties
	markDirtyOrders         func()
	loadBuyOrders           func(pair *PairV2, fromOrder *Limit, limit int) []uint32
	loadSellOrders          func(pair *PairV2, fromOrder *Limit, limit int) []uint32
	loadedSellOrders        *limits
	loadedBuyOrders         *limits
	unsortedDirtyBuyOrders  *orderDirties
	unsortedDirtySellOrders *orderDirties
	getLastTotalOrderID     func() uint32
	loadOrder               func(id uint32) *Limit
}

func (p *PairV2) GetID() uint32 {
	if p == nil {
		return 0
	}
	return *p.ID
}

func (p *PairV2) CalculateAddLiquidity(amount0 *big.Int, totalSupply *big.Int) (liquidity *big.Int, amount1 *big.Int) {
	reserve0, reserve1 := p.Reserves()
	return new(big.Int).Div(new(big.Int).Mul(totalSupply, amount0), reserve0), new(big.Int).Div(new(big.Int).Mul(amount0, reserve1), reserve0)
}

func (p *PairV2) Mint(amount0, amount1, totalSupply *big.Int) (liquidity *big.Int) {
	liquidity, amount1 = p.CalculateAddLiquidity(amount0, totalSupply)
	if liquidity.Sign() != 1 {
		panic(ErrorInsufficientLiquidityMinted)
	}
	p.update(amount0, amount1)
	return new(big.Int).Set(liquidity)
}

func (p *PairV2) Create(amount0, amount1 *big.Int) (liquidity *big.Int) {
	liquidity = startingSupply(amount0, amount1)

	if liquidity.Cmp(Bound) != 1 {
		panic(ErrorInsufficientLiquidityMinted)
	}
	p.update(amount0, amount1)
	return new(big.Int).Set(liquidity)
}

func (p *PairV2) CheckMint(amount0, maxAmount1, totalSupply *big.Int) (err error) {

	liquidity, amount1 := p.CalculateAddLiquidity(amount0, totalSupply)
	if amount1.Cmp(maxAmount1) == 1 {
		return ErrorInsufficientInputAmount
	}

	if liquidity.Sign() != 1 {
		return ErrorInsufficientLiquidityMinted
	}

	return nil
}
func (p *PairV2) CheckCreate(amount0, maxAmount1 *big.Int) (err error) {
	liquidity := startingSupply(amount0, maxAmount1)

	if liquidity.Cmp(Bound) != 1 {
		return ErrorInsufficientLiquidityMinted
	}

	return nil
}

func (p *PairV2) Burn(liquidity, minAmount0, minAmount1, totalSupply *big.Int) (amount0, amount1 *big.Int) {
	amount0, amount1 = p.Amounts(liquidity, totalSupply)

	if amount0.Cmp(minAmount0) == -1 || amount1.Cmp(minAmount1) == -1 {
		panic(ErrorInsufficientLiquidityBurned)
	}

	p.update(new(big.Int).Neg(amount0), new(big.Int).Neg(amount1))

	return amount0, amount1
}

func (p *PairV2) CheckBurn(liquidity, minAmount0, minAmount1, totalSupply *big.Int) error {
	if p == nil {
		return ErrorNotExist
	}
	amount0, amount1 := p.Amounts(liquidity, totalSupply)

	if amount0.Cmp(minAmount0) == -1 || amount1.Cmp(minAmount1) == -1 {
		return ErrorInsufficientLiquidityBurned
	}

	return nil
}

// Deprecated
func (p *PairV2) CalculateBuyForSellAllowNeg(amount0In *big.Int) (amount1Out *big.Int) {
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
func (p *PairV2) CalculateBuyForSell(amount0In *big.Int) (amount1Out *big.Int) {
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
func (p *PairV2) CalculateSellForBuyAllowNeg(amount1Out *big.Int) (amount0In *big.Int) {
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
func (p *PairV2) CalculateSellForBuy(amount1Out *big.Int) (amount0In *big.Int) {
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
func (p *PairV2) Swap(amount0In, amount1In, amount0Out, amount1Out *big.Int) (amount0, amount1 *big.Int) {
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

func (p *PairV2) checkSwap(amount0In, amount1In, amount0Out, amount1Out *big.Int) (err error) {
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

func (p *PairV2) update(amount0, amount1 *big.Int) {
	p.markDirty()

	p.pairData.Lock()
	defer p.pairData.Unlock()

	p.Reserve0.Add(p.Reserve0, amount0)
	p.Reserve1.Add(p.Reserve1, amount1)
}

func (p *PairV2) Amounts(liquidity, totalSupply *big.Int) (amount0 *big.Int, amount1 *big.Int) {
	p.pairData.RLock()
	defer p.pairData.RUnlock()
	amount0 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.Reserve0), totalSupply)
	amount1 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.Reserve1), totalSupply)
	return amount0, amount1
}
