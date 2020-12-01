package exchange

import (
	"errors"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/iavl"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
)

const minimumLiquidity int64 = 1000

type RSwap interface {
	Pairs() (pairs []pairKey)
	PairInfo(coinA, coinB types.CoinID) (totalSupply, reserve0, reserve1 *big.Int)
	PairExist(coinA, coinB types.CoinID) bool
	PairFromProvider(provider types.Address, coinA, coinB types.CoinID) (balance, amount0, amount1 *big.Int)
	CheckMint(coinA, coinB types.CoinID, amount0, amount1 *big.Int) error
	CheckBurn(address types.Address, coinA, coinB types.CoinID, liquidity *big.Int) error
}

type Swap struct {
	muPairs         sync.RWMutex
	pairs           map[pairKey]*Pair
	keyPairs        []pairKey
	isDirtyKeyPairs bool

	bus    *bus.Bus
	db     atomic.Value
	loaded bool
}

func New(bus *bus.Bus, db *iavl.ImmutableTree) *Swap {
	immutableTree := atomic.Value{}
	loaded := false
	if db != nil {
		immutableTree.Store(db)
	} else {
		loaded = true
	}
	return &Swap{pairs: map[pairKey]*Pair{}, bus: bus, db: immutableTree, loaded: loaded}
}

func (s *Swap) immutableTree() *iavl.ImmutableTree {
	db := s.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (s *Swap) Export(state *types.AppState) {
	panic("implement me")
}

var mainPrefix = "p"

type balance struct {
	address   types.Address
	liquidity *big.Int
}

type pairData struct {
	*sync.RWMutex
	reserve0    *big.Int
	reserve1    *big.Int
	totalSupply *big.Int
}

func (pd *pairData) TotalSupply() *big.Int {
	pd.RLock()
	defer pd.RUnlock()
	return new(big.Int).Set(pd.totalSupply)
}

func (pd *pairData) Reserves() (reserve0 *big.Int, reserve1 *big.Int) {
	pd.RLock()
	defer pd.RUnlock()
	return new(big.Int).Set(pd.reserve0), new(big.Int).Set(pd.reserve1)
}

func (pd *pairData) Revert() pairData {
	return pairData{
		RWMutex:     pd.RWMutex,
		reserve0:    pd.reserve1,
		reserve1:    pd.reserve0,
		totalSupply: pd.totalSupply,
	}
}

func (s *Swap) CheckBurn(address types.Address, coinA, coinB types.CoinID, liquidity *big.Int) error {
	return s.Pair(coinA, coinB).checkBurn(address, liquidity)
}
func (s *Swap) CheckMint(coinA, coinB types.CoinID, amount0, amount1 *big.Int) error {
	return s.Pair(coinA, coinB).checkMint(amount0, amount1)
}
func (s *Swap) Commit(db *iavl.MutableTree) error {
	basePath := []byte(mainPrefix)

	keyPairs := s.Pairs()
	s.muPairs.RLock()
	defer s.muPairs.RUnlock()

	if s.isDirtyKeyPairs {
		s.isDirtyKeyPairs = false
		pairsBytes, err := rlp.EncodeToBytes(keyPairs)
		if err != nil {
			return err
		}
		db.Set(basePath, pairsBytes)
	}

	for _, pairKey := range keyPairs {
		pair := s.pairs[pairKey]
		pairPath := append(basePath, pairKey.Bytes()...)

		if pair.isDirtyBalances {
			pair.isDirtyBalances = true
			var balances []*balance
			for address, liquidity := range pair.balances {
				if pair.balances[address].Sign() != 1 {
					delete(pair.balances, address)
					continue
				}
				balances = append(balances, &balance{address: address, liquidity: liquidity})
			}
			sort.Slice(balances, func(i, j int) bool {
				return balances[i].address.Compare(balances[j].address) == 1
			})
			balancesBytes, err := rlp.EncodeToBytes(balances)
			if err != nil {
				return err
			}
			db.Set(append(pairPath, 'b'), balancesBytes)
		}

		if !pair.isDirty {
			continue
		}
		pair.isDirty = false
		pairDataBytes, err := rlp.EncodeToBytes(pair.pairData)
		if err != nil {
			return err
		}
		db.Set(pairPath, pairDataBytes)
	}
	return nil
}

func (s *Swap) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	if s.immutableTree() == nil && s.loaded {
		s.loaded = false
	}
	s.db.Store(immutableTree)
}

func (s *Swap) Pairs() []pairKey {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	if !s.loaded {
		s.loaded = true
		_, value := s.immutableTree().Get([]byte(mainPrefix))
		if len(value) == 0 {
			return nil
		}
		err := rlp.DecodeBytes(value, &s.keyPairs)
		if err != nil {
			panic(err)
		}
		for _, keyPair := range s.keyPairs {
			if _, ok := s.pairs[keyPair]; ok {
				continue
			}
			s.pairs[keyPair] = nil
			s.pairs[keyPair.Revert()] = nil
		}
	}
	return s.keyPairs
}

func (s *Swap) pair(key pairKey) (*Pair, bool) {
	if key.isSorted() {
		pair, ok := s.pairs[key]
		return pair, ok
	}
	pair, ok := s.pairs[key.sort()]
	if !ok {
		return nil, false
	}
	return &Pair{
		muBalance: pair.muBalance,
		pairData:  pair.pairData.Revert(),
		balances:  pair.balances,
		dirty:     pair.dirty,
	}, true
}

func (s *Swap) PairExist(coinA, coinB types.CoinID) bool {
	return s.Pair(coinA, coinB) != nil
}
func (s *Swap) PairInfo(coinA, coinB types.CoinID) (totalSupply, reserve0, reserve1 *big.Int) {
	pair := s.Pair(coinA, coinB)
	if pair == nil {
		return nil, nil, nil
	}
	reserve0, reserve1 = pair.Reserves()
	totalSupply = pair.TotalSupply()
	return totalSupply, reserve0, reserve1
}

func (s *Swap) PairFromProvider(provider types.Address, coinA, coinB types.CoinID) (balance, amount0, amount1 *big.Int) {
	pair := s.Pair(coinA, coinB)
	if pair == nil {
		return nil, nil, nil
	}
	balance = pair.Balance(provider)
	if balance == nil {
		return nil, nil, nil
	}
	amount0, amount1 = pair.Amounts(balance)
	return balance, amount0, amount1
}

func (s *Swap) Pair(coinA, coinB types.CoinID) *Pair {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{CoinA: coinA, CoinB: coinB}
	pair, ok := s.pair(key)
	if pair != nil {
		return pair
	}

	if !ok && !s.loaded || ok && s.loaded {
		k := key.sort()
		pathPair := append([]byte(mainPrefix), k.Bytes()...)
		_, data := s.immutableTree().Get(pathPair)
		if len(data) == 0 {
			return nil
		}
		var pairData pairData
		err := rlp.DecodeBytes(data, &pairData)
		if err != nil {
			panic(err)
		}

		_, balancesBytes := s.immutableTree().Get(append(pathPair, 'b'))
		if len(balancesBytes) == 0 {
			return nil
		}
		var balances []*balance
		err = rlp.DecodeBytes(data, &balances)
		if err != nil {
			panic(err)
		}

		pairBalances := map[types.Address]*big.Int{}
		for _, balance := range balances {
			pairBalances[balance.address] = balance.liquidity
		}

		s.addPair(k, pairData, pairBalances)
	}
	pair, _ = s.pair(key)
	return pair
}

func (s *Swap) PairMint(address types.Address, coinA, coinB types.CoinID, amount0, amount1 *big.Int) (*big.Int, *big.Int) {
	pair := s.ReturnPair(coinA, coinB)
	liquidity := pair.Mint(address, amount0, amount1)
	return pair.Amounts(liquidity)
}

func (s *Swap) PairBurn(address types.Address, coinA, coinB types.CoinID, liquidity *big.Int) (*big.Int, *big.Int) {
	pair := s.Pair(coinA, coinB)
	return pair.Burn(address, liquidity)
}

type pairKey struct {
	CoinA, CoinB types.CoinID
}

func (pk pairKey) sort() pairKey {
	if pk.isSorted() {
		return pk
	}
	return pk.Revert()
}

func (pk pairKey) isSorted() bool {
	return pk.CoinA < pk.CoinB
}

func (pk pairKey) Revert() pairKey {
	return pairKey{CoinA: pk.CoinB, CoinB: pk.CoinA}
}

func (pk pairKey) Bytes() []byte {
	return append(pk.CoinA.Bytes(), pk.CoinB.Bytes()...)
}

var (
	ErrorIdenticalAddresses = errors.New("IDENTICAL_ADDRESSES")
	ErrorPairExists         = errors.New("PAIR_EXISTS")
)

func (s *Swap) ReturnPair(coinA, coinB types.CoinID) *Pair {
	if coinA == coinB {
		panic(ErrorIdenticalAddresses)
	}

	pair := s.Pair(coinA, coinB)
	if pair != nil {
		return pair
	}

	totalSupply, reserve0, reserve1, balances := big.NewInt(0), big.NewInt(0), big.NewInt(0), map[types.Address]*big.Int{}

	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{coinA, coinB}
	pair = s.addPair(key, pairData{reserve0: reserve0, reserve1: reserve1, totalSupply: totalSupply}, balances)
	s.addKeyPair(key)
	if !key.isSorted() {
		return &Pair{
			muBalance: pair.muBalance,
			pairData:  pair.Revert(),
			balances:  pair.balances,
			dirty:     pair.dirty,
		}
	}
	return pair
}

func (s *Swap) addPair(key pairKey, data pairData, balances map[types.Address]*big.Int) *Pair {
	if !key.isSorted() {
		key.Revert()
		data = data.Revert()
	}
	data.RWMutex = &sync.RWMutex{}
	pair := &Pair{
		muBalance: &sync.RWMutex{},
		pairData:  data,
		balances:  balances,
		dirty: &dirty{
			isDirty:         false,
			isDirtyBalances: false,
		},
	}
	s.pairs[key] = pair
	return pair
}

func (s *Swap) addKeyPair(key pairKey) {
	s.keyPairs = append(s.keyPairs, key.sort())
	s.isDirtyKeyPairs = true
}

var (
	ErrorInsufficientLiquidityMinted = errors.New("INSUFFICIENT_LIQUIDITY_MINTED")
)

type dirty struct {
	isDirty         bool
	isDirtyBalances bool
}
type Pair struct {
	pairData
	muBalance *sync.RWMutex
	balances  map[types.Address]*big.Int
	*dirty
}

func (p *Pair) Balance(address types.Address) (liquidity *big.Int) {
	p.muBalance.RLock()
	defer p.muBalance.RUnlock()

	balance := p.balances[address]
	if balance == nil {
		return nil
	}

	return new(big.Int).Set(balance)
}

func (p *Pair) Mint(address types.Address, amount0, amount1 *big.Int) (liquidity *big.Int) {
	totalSupply := p.TotalSupply()
	if totalSupply.Sign() != 1 {
		liquidity = startingSupply(amount0, amount1)
		p.mint(types.Address{}, big.NewInt(minimumLiquidity))
	} else {
		reserve0, reserve1 := p.Reserves()
		liquidity = new(big.Int).Div(new(big.Int).Mul(totalSupply, amount0), reserve0)
		liquidity1 := new(big.Int).Div(new(big.Int).Mul(totalSupply, amount1), reserve1)
		if liquidity.Cmp(liquidity1) == 1 {
			liquidity = liquidity1
		}
	}

	if liquidity.Sign() != 1 {
		panic(ErrorInsufficientLiquidityMinted)
	}

	p.mint(address, liquidity)
	p.update(amount0, amount1)

	return new(big.Int).Set(liquidity)
}

func (p *Pair) checkMint(amount0, amount1 *big.Int) (err error) {
	var liquidity *big.Int
	totalSupply := big.NewInt(0)
	if p != nil {
		totalSupply = p.TotalSupply()
	}
	if totalSupply.Sign() != 1 {
		liquidity = startingSupply(amount0, amount1)
	} else {
		reserve0, reserve1 := p.Reserves()
		liquidity = new(big.Int).Div(new(big.Int).Mul(totalSupply, amount0), reserve0)
		liquidity1 := new(big.Int).Div(new(big.Int).Mul(totalSupply, amount1), reserve1)
		if liquidity.Cmp(liquidity1) == 1 {
			liquidity = liquidity1
		}
	}

	if liquidity.Sign() != 1 {
		return ErrorInsufficientLiquidityMinted
	}

	return nil
}

var (
	ErrorInsufficientLiquidityBurned = errors.New("INSUFFICIENT_LIQUIDITY_BURNED")
)

func (p *Pair) Burn(address types.Address, liquidity *big.Int) (amount0 *big.Int, amount1 *big.Int) {
	balance := p.Balance(address)
	if balance == nil {
		panic(ErrorInsufficientLiquidityBurned)
	}

	if liquidity.Cmp(balance) == 1 {
		panic(ErrorInsufficientLiquidityBurned)
	}

	amount0, amount1 = p.Amounts(liquidity)

	if amount0.Sign() != 1 || amount1.Sign() != 1 {
		panic(ErrorInsufficientLiquidityBurned)
	}

	p.burn(address, liquidity)
	p.update(new(big.Int).Neg(amount0), new(big.Int).Neg(amount1))

	return amount0, amount1
}

func (p *Pair) checkBurn(address types.Address, liquidity *big.Int) (err error) {
	if p == nil {
		return errors.New("pair not found")
	}
	balance := p.Balance(address)
	if balance == nil {
		return ErrorInsufficientLiquidityBurned
	}

	if liquidity.Cmp(balance) == 1 {
		return ErrorInsufficientLiquidityBurned
	}

	amount0, amount1 := p.Amounts(liquidity)

	if amount0.Sign() != 1 || amount1.Sign() != 1 {
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

func (p *Pair) Swap(amount0In, amount1In, amount0Out, amount1Out *big.Int) (amount0, amount1 *big.Int, err error) {
	if amount0Out.Sign() != 1 && amount1Out.Sign() != 1 {
		return nil, nil, ErrorInsufficientOutputAmount
	}

	reserve0, reserve1 := p.Reserves()

	if amount0Out.Cmp(reserve0) == 1 || amount1Out.Cmp(reserve1) == 1 {
		return nil, nil, ErrorInsufficientLiquidity
	}

	amount0 = new(big.Int).Sub(amount0In, amount0Out)
	amount1 = new(big.Int).Sub(amount1In, amount1Out)

	if amount0.Sign() != 1 && amount1.Sign() != 1 {
		return nil, nil, ErrorInsufficientInputAmount
	}

	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(3)))
	balance1Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount1, reserve1), big.NewInt(1000)), new(big.Int).Mul(amount1In, big.NewInt(3)))

	if new(big.Int).Mul(balance0Adjusted, balance1Adjusted).Cmp(new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))) == -1 {
		return nil, nil, ErrorK
	}

	p.update(amount0, amount1)

	return amount0, amount1, nil
}

func (p *Pair) mint(address types.Address, value *big.Int) {
	p.pairData.Lock()
	defer p.pairData.Unlock()
	p.muBalance.Lock()
	defer p.muBalance.Unlock()

	p.isDirtyBalances = true
	p.isDirty = true
	p.totalSupply.Add(p.totalSupply, value)
	balance := p.balances[address]
	if balance == nil {
		p.balances[address] = big.NewInt(0)
	}
	p.balances[address].Add(p.balances[address], value)
}

func (p *Pair) burn(address types.Address, value *big.Int) {
	p.pairData.Lock()
	defer p.pairData.Unlock()
	p.muBalance.Lock()
	defer p.muBalance.Unlock()

	p.isDirtyBalances = true
	p.isDirty = true
	p.balances[address].Sub(p.balances[address], value)
	p.totalSupply.Sub(p.totalSupply, value)
}

func (p *Pair) update(amount0, amount1 *big.Int) {
	p.pairData.Lock()
	defer p.pairData.Unlock()

	p.isDirty = true
	p.reserve0.Add(p.reserve0, amount0)
	p.reserve1.Add(p.reserve1, amount1)
}

func (p *Pair) Amounts(liquidity *big.Int) (amount0 *big.Int, amount1 *big.Int) {
	p.pairData.RLock()
	defer p.pairData.RUnlock()
	amount0 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.reserve0), p.totalSupply)
	amount1 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.reserve1), p.totalSupply)
	return amount0, amount1
}

func startingSupply(amount0 *big.Int, amount1 *big.Int) *big.Int {
	mul := new(big.Int).Mul(amount0, amount1)
	sqrt := new(big.Int).Sqrt(mul)
	return new(big.Int).Sub(sqrt, big.NewInt(minimumLiquidity))
}
