package swap

import (
	"errors"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/iavl"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

const minimumLiquidity int64 = 1000

type RSwap interface {
	SwapPool(coin0, coin1 types.CoinID) (totalSupply, reserve0, reserve1 *big.Int)
	SwapPoolExist(coin0, coin1 types.CoinID) bool
	SwapPoolFromProvider(provider types.Address, coin0, coin1 types.CoinID) (balance, amount0, amount1 *big.Int)
	CheckMint(coin0, coin1 types.CoinID, amount0, amount1 *big.Int) error
	CheckBurn(address types.Address, coin0, coin1 types.CoinID, liquidity *big.Int) error
	CheckSwap(coin0, coin1 types.CoinID, amount0In, amount1Out *big.Int) error
	Export(state *types.AppState)
}

type Swap struct {
	muPairs sync.RWMutex
	pairs   map[pairKey]*Pair

	bus *bus.Bus
	db  atomic.Value
}

func New(bus *bus.Bus, db *iavl.ImmutableTree) *Swap {
	immutableTree := atomic.Value{}
	immutableTree.Store(db)
	return &Swap{pairs: map[pairKey]*Pair{}, bus: bus, db: immutableTree}
}

func (s *Swap) immutableTree() *iavl.ImmutableTree {
	return s.db.Load().(*iavl.ImmutableTree)
}

func (s *Swap) Export(state *types.AppState) {
	s.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		coin0 := types.BytesToCoinID(key[1:5])
		coin1 := types.BytesToCoinID(key[5:9])
		pair := s.ReturnPair(coin0, coin1)
		if len(key) > 9 {
			provider := types.BytesToAddress(key[9:])
			pair.balances[provider] = pair.loadBalance(provider)
		}

		return false
	})

	for key, pair := range s.pairs {
		reserve0, reserve1 := pair.Reserves()
		swap := types.Swap{
			Providers:   make([]types.BalanceProvider, 0, len(pair.balances)),
			Coin0:       uint64(key.Coin0),
			Coin1:       uint64(key.Coin1),
			Reserve0:    reserve0.String(),
			Reserve1:    reserve1.String(),
			TotalSupply: pair.GetTotalSupply().String(),
		}

		for address, balance := range pair.balances {
			swap.Providers = append(swap.Providers, types.BalanceProvider{
				Address:   address,
				Liquidity: balance.Liquidity.String(),
			})
		}

		sort.Slice(swap.Providers, func(i, j int) bool {
			return swap.Providers[i].Address.Compare(swap.Providers[j].Address) == -1
		})

		state.Swap = append(state.Swap, swap)
	}

	sort.Slice(state.Swap, func(i, j int) bool {
		return strconv.Itoa(int(state.Swap[i].Coin0))+"-"+strconv.Itoa(int(state.Swap[i].Coin1)) < strconv.Itoa(int(state.Swap[j].Coin0))+"-"+strconv.Itoa(int(state.Swap[j].Coin1))
	})
}

func (s *Swap) Import(state *types.AppState) {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()
	for _, swap := range state.Swap {
		pair := s.ReturnPair(types.CoinID(swap.Coin0), types.CoinID(swap.Coin1))
		pair.TotalSupply.Set(helpers.StringToBigInt(swap.TotalSupply))
		pair.Reserve0.Set(helpers.StringToBigInt(swap.Reserve0))
		pair.Reserve1.Set(helpers.StringToBigInt(swap.Reserve1))
		pair.isDirty = true
		for _, provider := range swap.Providers {
			pair.balances[provider.Address] = &Balance{Liquidity: helpers.StringToBigInt(provider.Liquidity), isDirty: true}
		}
	}
}

const mainPrefix = byte('s')

type dirty struct{ isDirty bool }

type pairData struct {
	*sync.RWMutex
	Reserve0    *big.Int
	Reserve1    *big.Int
	TotalSupply *big.Int
	*dirty
}

func (pd *pairData) GetTotalSupply() *big.Int {
	pd.RLock()
	defer pd.RUnlock()
	return new(big.Int).Set(pd.TotalSupply)
}

func (pd *pairData) Reserves() (reserve0 *big.Int, reserve1 *big.Int) {
	pd.RLock()
	defer pd.RUnlock()
	return new(big.Int).Set(pd.Reserve0), new(big.Int).Set(pd.Reserve1)
}

func (pd *pairData) Revert() *pairData {
	return &pairData{
		RWMutex:     pd.RWMutex,
		Reserve0:    pd.Reserve1,
		Reserve1:    pd.Reserve0,
		TotalSupply: pd.TotalSupply,
		dirty:       pd.dirty,
	}
}

func (s *Swap) CheckMint(coinA, coinB types.CoinID, amount0, amount1 *big.Int) error {
	return s.Pair(coinA, coinB).checkMint(amount0, amount1)
}
func (s *Swap) CheckBurn(address types.Address, coinA, coinB types.CoinID, liquidity *big.Int) error {
	return s.Pair(coinA, coinB).checkBurn(address, liquidity)
}
func (s *Swap) CheckSwap(coinA, coinB types.CoinID, amount0In, amount1Out *big.Int) error {
	return s.Pair(coinA, coinB).checkSwap(amount0In, big.NewInt(0), big.NewInt(0), amount1Out)
}

func (s *Swap) Commit(db *iavl.MutableTree) error {
	basePath := []byte{mainPrefix}

	s.muPairs.RLock()
	defer s.muPairs.RUnlock()

	for key, pair := range s.pairs {
		if !pair.isDirty {
			continue
		}

		pairPath := append(basePath, key.Bytes()...)

		pair.isDirty = false
		pairDataBytes, err := rlp.EncodeToBytes(pair.pairData)
		if err != nil {
			return err
		}
		db.Set(pairPath, pairDataBytes)

		for address, balance := range pair.balances {
			if !balance.isDirty {
				continue
			}
			balance.isDirty = false
			balanceBytes, err := rlp.EncodeToBytes(balance)
			if err != nil {
				return err
			}
			db.Set(append(pairPath, address.Bytes()...), balanceBytes)
		}

	}
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
	if !ok {
		return nil, false
	}
	if key.isSorted() {
		return pair, true
	}
	return &Pair{
		muBalances:  pair.muBalances,
		pairData:    pair.pairData.Revert(),
		balances:    pair.balances,
		loadBalance: pair.loadBalance,
	}, true
}

func (s *Swap) SwapPool(coinA, coinB types.CoinID) (totalSupply, reserve0, reserve1 *big.Int) {
	pair := s.Pair(coinA, coinB)
	if pair == nil {
		return nil, nil, nil
	}
	reserve0, reserve1 = pair.Reserves()
	totalSupply = pair.GetTotalSupply()
	return totalSupply, reserve0, reserve1
}

func (s *Swap) SwapPoolFromProvider(provider types.Address, coinA, coinB types.CoinID) (balance, amount0, amount1 *big.Int) {
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

func (s *Swap) Pair(coin0, coin1 types.CoinID) *Pair {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	key := pairKey{Coin0: coin0, Coin1: coin1}
	pair, ok := s.pair(key)
	if ok {
		return pair
	}

	pathPair := append([]byte{mainPrefix}, key.sort().Bytes()...)
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

	return pair
}

func (s *Swap) PairMint(address types.Address, coin0, coin1 types.CoinID, amount0, amount1 *big.Int) (*big.Int, *big.Int) {
	pair := s.ReturnPair(coin0, coin1)
	oldReserve0, oldReserve1 := pair.Reserves()
	_ = pair.Mint(address, amount0, amount1)
	newReserve0, newReserve1 := pair.Reserves()

	balance0 := new(big.Int).Sub(newReserve0, oldReserve0)
	balance1 := new(big.Int).Sub(newReserve1, oldReserve1)

	s.bus.Checker().AddCoin(coin0, balance0)
	s.bus.Checker().AddCoin(coin1, balance1)

	return balance0, balance1
}

func (s *Swap) PairBurn(address types.Address, coin0, coin1 types.CoinID, liquidity *big.Int) (*big.Int, *big.Int) {
	pair := s.Pair(coin0, coin1)
	oldReserve0, oldReserve1 := pair.Reserves()
	_, _ = pair.Burn(address, liquidity)
	newReserve0, newReserve1 := pair.Reserves()

	balance0 := new(big.Int).Sub(oldReserve0, newReserve0)
	balance1 := new(big.Int).Sub(oldReserve1, newReserve1)

	s.bus.Checker().AddCoin(coin0, new(big.Int).Neg(balance0))
	s.bus.Checker().AddCoin(coin1, new(big.Int).Neg(balance1))

	return balance0, balance1
}

func (s *Swap) PairSwap(coin0, coin1 types.CoinID, amount0In, amount1Out *big.Int) (*big.Int, *big.Int) {
	pair := s.Pair(coin0, coin1)
	balance0, balance1 := pair.Swap(amount0In, amount1Out)
	s.bus.Checker().AddCoin(coin0, balance0)
	s.bus.Checker().AddCoin(coin1, new(big.Int).Neg(balance1))
	return balance0, balance1
}

type pairKey struct {
	Coin0, Coin1 types.CoinID
}

func (pk pairKey) sort() pairKey {
	if pk.isSorted() {
		return pk
	}
	return pk.Revert()
}

func (pk *pairKey) isSorted() bool {
	return pk.Coin0 < pk.Coin1
}

func (pk *pairKey) Revert() pairKey {
	return pairKey{Coin0: pk.Coin1, Coin1: pk.Coin0}
}

func (pk pairKey) Bytes() []byte {
	return append(pk.Coin0.Bytes(), pk.Coin1.Bytes()...)
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
		return &Pair{
			muBalances:  pair.muBalances,
			pairData:    pair.Revert(),
			balances:    pair.balances,
			loadBalance: pair.loadBalance,
		}
	}
	return pair
}

func (s *Swap) loadBalanceFunc(key *pairKey) func(address types.Address) *Balance {
	return func(address types.Address) *Balance {
		_, balancesBytes := s.immutableTree().Get(append(append([]byte{mainPrefix}, key.Bytes()...), address.Bytes()...))
		if len(balancesBytes) == 0 {
			return nil
		}

		balance := new(Balance)
		if err := rlp.DecodeBytes(balancesBytes, balance); err != nil {
			panic(err)
		}

		return balance
	}
}

func (s *Swap) addPair(key pairKey) *Pair {
	data := &pairData{
		RWMutex:     &sync.RWMutex{},
		Reserve0:    big.NewInt(0),
		Reserve1:    big.NewInt(0),
		TotalSupply: big.NewInt(0),
		dirty:       &dirty{},
	}
	if !key.isSorted() {
		key.Revert()
		data = data.Revert()
	}
	pair := &Pair{
		muBalances:  &sync.RWMutex{},
		pairData:    data,
		balances:    map[types.Address]*Balance{},
		loadBalance: s.loadBalanceFunc(&key),
	}
	s.pairs[key] = pair
	return pair
}

var (
	ErrorInsufficientLiquidityMinted = errors.New("INSUFFICIENT_LIQUIDITY_MINTED")
)

type Balance struct {
	Liquidity *big.Int
	isDirty   bool
}

type Pair struct {
	*pairData
	muBalances  *sync.RWMutex
	balances    map[types.Address]*Balance
	loadBalance func(address types.Address) *Balance
}

func (p *Pair) Balance(address types.Address) (liquidity *big.Int) {
	p.muBalances.RLock()
	balance, ok := p.balances[address]
	p.muBalances.RUnlock()
	if ok {
		if balance == nil {
			return nil
		}
		return new(big.Int).Set(balance.Liquidity)
	}

	p.muBalances.Lock()
	defer p.muBalances.Unlock()

	p.balances[address] = p.loadBalance(address)

	if p.balances[address] == nil {
		return nil
	}

	return new(big.Int).Set(p.balances[address].Liquidity)
}

func (p *Pair) liquidity(amount0, amount1 *big.Int) (liquidity, a, b *big.Int) {
	totalSupply := p.GetTotalSupply()
	reserve0, reserve1 := p.Reserves()
	liquidity = new(big.Int).Div(new(big.Int).Mul(totalSupply, amount0), reserve0)
	liquidity1 := new(big.Int).Div(new(big.Int).Mul(totalSupply, amount1), reserve1)
	if liquidity.Cmp(liquidity1) == 1 {
		liquidity = liquidity1
		amount0 = new(big.Int).Div(new(big.Int).Mul(liquidity, reserve0), totalSupply)
	} else {
		amount1 = new(big.Int).Div(new(big.Int).Mul(liquidity, reserve1), totalSupply)
	}
	return liquidity, amount0, amount1
}

func (p *Pair) Mint(address types.Address, amount0, amount1 *big.Int) (liquidity *big.Int) {
	totalSupply := p.GetTotalSupply()
	if totalSupply.Sign() != 1 {
		liquidity = startingSupply(amount0, amount1)
		p.mint(types.Address{}, big.NewInt(minimumLiquidity))
	} else {
		liquidity, amount0, amount1 = p.liquidity(amount0, amount1)
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
		totalSupply = p.GetTotalSupply()
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

func (p *Pair) Swap(amount0In, amount1Out *big.Int) (amount0, amount1 *big.Int) {
	reserve0, reserve1 := p.Reserves()

	if amount1Out.Cmp(reserve1) == 1 {
		panic(ErrorInsufficientLiquidity)
	}

	if amount0In.Sign() == -1 {
		panic(ErrorInsufficientInputAmount)
	}

	kAdjusted := new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))
	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0In, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(3)))

	amount1 = new(big.Int).Sub(new(big.Int).Sub(reserve1, new(big.Int).Quo(kAdjusted, new(big.Int).Mul(balance0Adjusted, big.NewInt(1000)))), big.NewInt(1))

	if amount1Out.Cmp(amount1) == 1 {
		panic(ErrorK)
	}

	p.update(amount0In, new(big.Int).Neg(amount1))

	return amount0In, amount1
}

func (p *Pair) checkSwap(amount0In, amount1In, amount0Out, amount1Out *big.Int) (err error) {
	if amount0Out.Sign() != 1 && amount1Out.Sign() != 1 {
		return ErrorInsufficientOutputAmount
	}

	reserve0, reserve1 := p.Reserves()

	if amount0Out.Cmp(reserve0) == 1 || amount1Out.Cmp(reserve1) == 1 {
		return ErrorInsufficientLiquidity
	}

	amount0 := new(big.Int).Sub(amount0In, amount0Out)
	amount1 := new(big.Int).Sub(amount1In, amount1Out)

	if amount0.Sign() != 1 && amount1.Sign() != 1 {
		return ErrorInsufficientInputAmount
	}

	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0, reserve0), big.NewInt(1000)), new(big.Int).Mul(amount0In, big.NewInt(3)))
	balance1Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount1, reserve1), big.NewInt(1000)), new(big.Int).Mul(amount1In, big.NewInt(3)))

	if new(big.Int).Mul(balance0Adjusted, balance1Adjusted).Cmp(new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))) == -1 {
		return ErrorK
	}
	return nil
}

func (p *Pair) mint(address types.Address, value *big.Int) {
	p.pairData.Lock()
	defer p.pairData.Unlock()

	p.isDirty = true
	p.TotalSupply.Add(p.TotalSupply, value)

	p.muBalances.Lock()
	defer p.muBalances.Unlock()

	balance := p.balances[address]
	if balance == nil {
		p.balances[address] = &Balance{Liquidity: big.NewInt(0)}
	}
	p.balances[address].Liquidity.Add(p.balances[address].Liquidity, value)
	p.balances[address].isDirty = true
}

func (p *Pair) burn(address types.Address, value *big.Int) {
	p.pairData.Lock()
	defer p.pairData.Unlock()

	p.isDirty = true
	p.TotalSupply.Sub(p.TotalSupply, value)

	p.muBalances.Lock()
	defer p.muBalances.Unlock()

	p.balances[address].isDirty = true
	p.balances[address].Liquidity.Sub(p.balances[address].Liquidity, value)
}

func (p *Pair) update(amount0, amount1 *big.Int) {
	p.pairData.Lock()
	defer p.pairData.Unlock()

	p.isDirty = true
	p.Reserve0.Add(p.Reserve0, amount0)
	p.Reserve1.Add(p.Reserve1, amount1)
}

func (p *Pair) Amounts(liquidity *big.Int) (amount0 *big.Int, amount1 *big.Int) {
	p.pairData.RLock()
	defer p.pairData.RUnlock()
	amount0 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.Reserve0), p.TotalSupply)
	amount1 = new(big.Int).Div(new(big.Int).Mul(liquidity, p.Reserve1), p.TotalSupply)
	return amount0, amount1
}

func (p *Pair) BoundedAmounts() (amount0 *big.Int, amount1 *big.Int) {
	boundedSupply := p.Balance(types.Address{})
	return p.Amounts(boundedSupply)
}

func startingSupply(amount0 *big.Int, amount1 *big.Int) *big.Int {
	mul := new(big.Int).Mul(amount0, amount1)
	sqrt := new(big.Int).Sqrt(mul)
	return new(big.Int).Sub(sqrt, big.NewInt(minimumLiquidity))
}
