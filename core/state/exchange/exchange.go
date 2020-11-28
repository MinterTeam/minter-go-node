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
	Pair(coinA, coinB types.CoinID) (*Pair, error)
	Pairs() (pairs []pairKey, err error)
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
	return pd.totalSupply
}

func (pd *pairData) Reserves() (reserve0 *big.Int, reserve1 *big.Int) {
	pd.RLock()
	defer pd.RUnlock()
	return pd.reserve0, pd.reserve1
}

func (pd *pairData) Revert() pairData {
	return pairData{
		RWMutex:     pd.RWMutex,
		reserve0:    pd.reserve1,
		reserve1:    pd.reserve0,
		totalSupply: pd.totalSupply,
	}
}

func (s *Swap) Commit(db *iavl.MutableTree) error {
	basePath := []byte(mainPrefix)

	keyPairs, err := s.Pairs()
	if err != nil {
		return err
	}
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

func (s *Swap) Pairs() ([]pairKey, error) {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	if !s.loaded {
		s.loaded = true
		_, value := s.immutableTree().Get([]byte(mainPrefix))
		if len(value) == 0 {
			return nil, nil
		}
		err := rlp.DecodeBytes(value, &s.keyPairs)
		if err != nil {
			return nil, err
		}
		for _, keyPair := range s.keyPairs {
			if _, ok := s.pairs[keyPair]; ok {
				continue
			}
			s.pairs[keyPair] = nil
			s.pairs[keyPair.Revert()] = nil
		}
	}
	return s.keyPairs, nil
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
		pairData: pair.pairData,
		balances: pair.balances,
		dirty:    pair.dirty,
	}, true
}

func (s *Swap) Pair(coinA, coinB types.CoinID) (*Pair, error) {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{CoinA: coinA, CoinB: coinB}
	pair, ok := s.pair(key)
	if pair != nil {
		return pair, nil
	}

	if !ok && !s.loaded || ok && s.loaded {
		k := key.sort()
		pathPair := append([]byte(mainPrefix), k.Bytes()...)
		_, data := s.immutableTree().Get(pathPair)
		if len(data) == 0 {
			return nil, nil
		}
		var pairData pairData
		err := rlp.DecodeBytes(data, &pairData)
		if err != nil {
			return nil, err
		}

		_, balancesBytes := s.immutableTree().Get(append(pathPair, 'b'))
		if len(balancesBytes) == 0 {
			return nil, nil
		}
		var balances []*balance
		err = rlp.DecodeBytes(data, &balances)
		if err != nil {
			return nil, err
		}

		pairBalances := map[types.Address]*big.Int{}
		for _, balance := range balances {
			pairBalances[balance.address] = balance.liquidity
		}

		s.addPair(k, pairData, pairBalances)
	}
	pair, _ = s.pair(key)
	return pair, nil
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
	return pk.CoinA < pk.CoinA
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

func (s *Swap) CreatePair(coinA, coinB types.CoinID) (*Pair, error) {
	if coinA == coinB {
		return nil, ErrorIdenticalAddresses
	}

	pair, err := s.Pair(coinA, coinB)
	if err != nil {
		return nil, err
	}
	if pair != nil {
		return nil, ErrorPairExists
	}

	totalSupply, reserve0, reserve1, balances := big.NewInt(0), big.NewInt(0), big.NewInt(0), map[types.Address]*big.Int{}

	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{coinA, coinB}
	pair = s.addPair(key, pairData{reserve0: reserve0, reserve1: reserve1, totalSupply: totalSupply}, balances)
	s.addKeyPair(key)
	if !key.isSorted() {
		return &Pair{
			pairData: pair.Revert(),
			balances: pair.balances,
			dirty:    pair.dirty,
		}, nil
	}
	return pair, nil
}

func (s *Swap) addPair(key pairKey, data pairData, balances map[types.Address]*big.Int) *Pair {
	if !key.isSorted() {
		key.Revert()
		data = data.Revert()
	}
	data.RWMutex = &sync.RWMutex{}
	pair := &Pair{
		pairData: data,
		balances: balances,
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
	muBalance sync.RWMutex
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

func (p *Pair) Mint(address types.Address, amount0, amount1 *big.Int) (liquidity *big.Int, err error) {
	if p.TotalSupply().Sign() == 0 {
		liquidity = startingSupply(amount0, amount1)
		if liquidity.Sign() != 1 {
			return nil, ErrorInsufficientLiquidityMinted
		}
		p.mint(types.Address{}, big.NewInt(minimumLiquidity))
	} else {
		liquidity := new(big.Int).Div(new(big.Int).Mul(p.totalSupply, amount0), p.reserve0)
		liquidity1 := new(big.Int).Div(new(big.Int).Mul(p.totalSupply, amount1), p.reserve1)
		if liquidity.Cmp(liquidity1) == 1 {
			liquidity = liquidity1
		}
	}

	p.mint(address, liquidity)
	p.update(amount0, amount1)

	return liquidity, nil
}

var (
	ErrorInsufficientLiquidityBurned = errors.New("INSUFFICIENT_LIQUIDITY_BURNED")
)

func (p *Pair) Burn(address types.Address, liquidity *big.Int) (amount0 *big.Int, amount1 *big.Int, err error) {
	balance := p.Balance(address)
	if balance == nil {
		return nil, nil, ErrorInsufficientLiquidityBurned
	}

	if liquidity.Cmp(balance) == 1 {
		return nil, nil, ErrorInsufficientLiquidityBurned
	}

	amount0, amount1 = p.Amounts(liquidity)

	if amount0.Sign() != 1 || amount1.Sign() != 1 {
		return nil, nil, ErrorInsufficientLiquidityBurned
	}

	p.burn(address, liquidity)
	p.update(new(big.Int).Neg(amount0), new(big.Int).Neg(amount1))

	return amount0, amount1, nil
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
