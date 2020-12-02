package exchange

import (
	"errors"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/iavl"
	"math/big"
	"sync"
	"sync/atomic"
)

const minimumLiquidity int64 = 1000

type RSwap interface {
	PairInfo(coin types.CoinID) (totalSupply, reserve0, reserve1 *big.Int)
	PairExist(coin types.CoinID) bool
	PairFromProvider(provider types.Address, coin types.CoinID) (balance, amount0, amount1 *big.Int)
	CheckMint(coin types.CoinID, amount0, amount1 *big.Int) error
	CheckBurn(address types.Address, coin types.CoinID, liquidity *big.Int) error
	Export(state *types.AppState)
}

type Swap struct {
	muPairs sync.RWMutex
	pairs   map[types.CoinID]*Pair

	bus *bus.Bus
	db  atomic.Value
}

func New(bus *bus.Bus, db *iavl.ImmutableTree) *Swap {
	immutableTree := atomic.Value{}
	immutableTree.Store(db)
	return &Swap{pairs: map[types.CoinID]*Pair{}, bus: bus, db: immutableTree}
}

func (s *Swap) immutableTree() *iavl.ImmutableTree {
	db := s.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (s *Swap) Export(state *types.AppState) {
	s.immutableTree().Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix[0] {
			coin := types.BytesToCoinID(key[1:5])
			pair := s.Pair(coin)
			if len(key) > 5 {
				types.BytesToAddress(key[5:])
			}
			_ = pair
		}
		return false
	})
	// todo: iterateRange
	// for _, key := range s.Pairs() {
	// 	pair := s.Pair(key.CoinA, key.CoinB)
	// 	if pair == nil {
	// 		continue
	// 	}
	//
	// 	balances := pair.Balances()
	// 	reserve0, reserve1 := pair.Reserves()
	// 	swap := types.Swap{
	// 		Providers:   make([]types.BalanceProvider, 0, len(balances)),
	// 		Coin0:       uint64(key.CoinA),
	// 		Coin:        uint64(key.CoinB),
	// 		ReserveBip:    reserve0.String(),
	// 		ReserveCustom:    reserve1.String(),
	// 		TotalSupply: pair.GetTotalSupply().String(),
	// 	}
	//
	// 	for _, balance := range balances {
	// 		swap.Providers = append(swap.Providers, types.BalanceProvider{
	// 			Address:   balance.Address,
	// 			Liquidity: balance.Liquidity.String(),
	// 		})
	// 	}
	//
	// 	state.Swap = append(state.Swap, swap)
	// }
}

func (s *Swap) Import(state *types.AppState) {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()
	for _, swap := range state.Swap {
		pair := s.ReturnPair(types.CoinID(swap.Coin))
		pair.TotalSupply.Set(helpers.StringToBigInt(swap.TotalSupply))
		pair.ReserveBip.Set(helpers.StringToBigInt(swap.ReserveBip))
		pair.ReserveCustom.Set(helpers.StringToBigInt(swap.ReserveCustom))
		pair.isDirty = true
		for _, provider := range swap.Providers {
			pair.balances[provider.Address] = &Balance{Liquidity: helpers.StringToBigInt(provider.Liquidity), isDirty: true}
		}
	}

}

var mainPrefix = "s"

func (p *Pair) GetTotalSupply() *big.Int {
	p.RLock()
	defer p.RUnlock()
	return new(big.Int).Set(p.TotalSupply)
}

func (p *Pair) Reserves() (reserve0 *big.Int, reserve1 *big.Int) {
	p.RLock()
	defer p.RUnlock()
	return new(big.Int).Set(p.ReserveBip), new(big.Int).Set(p.ReserveCustom)
}

func (s *Swap) CheckBurn(address types.Address, coin types.CoinID, liquidity *big.Int) error {
	return s.Pair(coin).checkBurn(address, liquidity)
}
func (s *Swap) CheckMint(coin types.CoinID, amount0, amount1 *big.Int) error {
	return s.Pair(coin).checkMint(amount0, amount1)
}

func (s *Swap) Commit(db *iavl.MutableTree) error {
	basePath := []byte(mainPrefix)

	s.muPairs.RLock()
	defer s.muPairs.RUnlock()

	for coin, pair := range s.pairs {
		if !pair.isDirty {
			continue
		}

		pairPath := append(basePath, coin.Bytes()...)

		pair.isDirty = false
		pairDataBytes, err := rlp.EncodeToBytes(pair)
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

func (s *Swap) PairExist(coin types.CoinID) bool {
	return s.Pair(coin) != nil
}
func (s *Swap) PairInfo(coin types.CoinID) (totalSupply, reserve0, reserve1 *big.Int) {
	pair := s.Pair(coin)
	if pair == nil {
		return nil, nil, nil
	}

	reserve0, reserve1 = pair.Reserves()
	totalSupply = pair.GetTotalSupply()
	return totalSupply, reserve0, reserve1
}

func (s *Swap) PairFromProvider(provider types.Address, coin types.CoinID) (balance, amountBip, amountCustom *big.Int) {
	pair := s.Pair(coin)
	if pair == nil {
		return nil, nil, nil
	}

	balance = pair.Balance(provider)
	if balance == nil {
		return nil, nil, nil
	}

	amountBip, amountCustom = pair.Amounts(balance)
	return balance, amountBip, amountCustom
}

func (s *Swap) Pair(coin types.CoinID) *Pair {
	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	pair, ok := s.pairs[coin]
	if ok {
		return pair
	}

	pathPair := append([]byte(mainPrefix), coin.Bytes()...)
	_, data := s.immutableTree().Get(pathPair)
	if len(data) == 0 {
		s.pairs[coin] = nil
		return nil
	}

	pair = new(Pair)
	err := rlp.DecodeBytes(data, &pair)
	if err != nil {
		panic(err)
	}

	pair.loadBalance = func(address types.Address) *Balance {
		_, balancesBytes := s.immutableTree().Get(append(pathPair, address.Bytes()...))
		if len(balancesBytes) == 0 {
			pair.balances[address] = nil
			return nil
		}

		balance := new(Balance)
		err = rlp.DecodeBytes(balancesBytes, balance)
		if err != nil {
			panic(err)
		}

		return balance
	}

	s.pairs[coin] = pair
	return pair
}

func (s *Swap) PairMint(address types.Address, custom types.CoinID, amountBip, amountCustom *big.Int) (*big.Int, *big.Int) {
	pair := s.ReturnPair(custom)
	oldReserveBip, oldReserveCustom := pair.Reserves()
	_ = pair.Mint(address, amountBip, amountCustom)
	newReserveBip, newReserveCustom := pair.Reserves()

	balanceBip := new(big.Int).Sub(newReserveBip, oldReserveBip)
	balanceCustom := new(big.Int).Sub(newReserveCustom, oldReserveCustom)

	s.bus.Checker().AddCoin(types.GetBaseCoinID(), balanceBip)
	s.bus.Checker().AddCoin(custom, balanceCustom)

	return balanceBip, balanceCustom
}

func (s *Swap) PairBurn(address types.Address, custom types.CoinID, liquidity *big.Int) (*big.Int, *big.Int) {
	pair := s.Pair(custom)
	oldReserveBip, oldReserveCustom := pair.Reserves()
	_, _ = pair.Burn(address, liquidity)
	newReserveBip, newReserveCustom := pair.Reserves()

	balanceBip := new(big.Int).Sub(oldReserveBip, newReserveBip)
	balanceCustom := new(big.Int).Sub(oldReserveCustom, newReserveCustom)

	s.bus.Checker().AddCoin(types.GetBaseCoinID(), new(big.Int).Neg(balanceBip))
	s.bus.Checker().AddCoin(custom, new(big.Int).Neg(balanceCustom))

	return balanceBip, balanceCustom
}

var (
	ErrorIdenticalAddresses = errors.New("IDENTICAL_ADDRESSES")
)

func (s *Swap) ReturnPair(coin types.CoinID) *Pair {
	if coin == types.GetBaseCoinID() {
		panic(ErrorIdenticalAddresses)
	}

	pair := s.Pair(coin)
	if pair != nil {
		return pair
	}

	s.muPairs.Lock()
	defer s.muPairs.Unlock()

	return s.addPair(coin)
}

func (s *Swap) addPair(coin types.CoinID) *Pair {
	pair := &Pair{
		ReserveBip:    big.NewInt(0),
		ReserveCustom: big.NewInt(0),
		TotalSupply:   big.NewInt(0),
		balances:      map[types.Address]*Balance{},
	}
	s.pairs[coin] = pair
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
	sync.RWMutex
	ReserveBip    *big.Int
	ReserveCustom *big.Int
	TotalSupply   *big.Int
	isDirty       bool

	muBalances  sync.RWMutex
	loadBalance func(address types.Address) *Balance
	balances    map[types.Address]*Balance
}

func (p *Pair) Balance(address types.Address) (liquidity *big.Int) {
	p.muBalances.Lock()
	defer p.muBalances.Unlock()

	balance, ok := p.balances[address]
	if ok {
		if balance == nil {
			return nil
		}
		return new(big.Int).Set(balance.Liquidity)
	}

	p.balances[address] = p.loadBalance(address)

	return new(big.Int).Set(balance.Liquidity)
}

func (p *Pair) liquidity(amountBip, amountCustom *big.Int) (liquidity, a, b *big.Int) {
	totalSupply := p.GetTotalSupply()
	reserveBip, reserveCustom := p.Reserves()
	liquidity = new(big.Int).Div(new(big.Int).Mul(totalSupply, amountBip), reserveBip)
	liquidity1 := new(big.Int).Div(new(big.Int).Mul(totalSupply, amountCustom), reserveCustom)
	if liquidity.Cmp(liquidity1) == 1 {
		liquidity = liquidity1
		amountBip = new(big.Int).Div(new(big.Int).Mul(liquidity, reserveBip), totalSupply)
	} else {
		amountCustom = new(big.Int).Div(new(big.Int).Mul(liquidity, reserveCustom), totalSupply)
	}
	return liquidity, amountBip, amountCustom
}

func (p *Pair) Mint(address types.Address, amountBip, amountCustom *big.Int) (liquidity *big.Int) {
	totalSupply := p.GetTotalSupply()
	if totalSupply.Sign() != 1 {
		liquidity = startingSupply(amountBip, amountCustom)
		p.mint(types.Address{}, big.NewInt(minimumLiquidity))
	} else {
		liquidity, amountBip, amountCustom = p.liquidity(amountBip, amountCustom)
	}

	if liquidity.Sign() != 1 {
		panic(ErrorInsufficientLiquidityMinted)
	}

	p.mint(address, liquidity)
	p.update(amountBip, amountCustom)

	return new(big.Int).Set(liquidity)
}

func (p *Pair) checkMint(amountBip, amountCustom *big.Int) (err error) {
	var liquidity *big.Int
	totalSupply := big.NewInt(0)
	if p != nil {
		totalSupply = p.GetTotalSupply()
	}
	if totalSupply.Sign() != 1 {
		liquidity = startingSupply(amountBip, amountCustom)
	} else {
		reserveBip, reserveCustom := p.Reserves()
		liquidity = new(big.Int).Div(new(big.Int).Mul(totalSupply, amountBip), reserveBip)
		liquidity1 := new(big.Int).Div(new(big.Int).Mul(totalSupply, amountCustom), reserveCustom)
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

func (p *Pair) Burn(address types.Address, liquidity *big.Int) (amountBip *big.Int, amountCustom *big.Int) {
	balance := p.Balance(address)
	if balance == nil {
		panic(ErrorInsufficientLiquidityBurned)
	}

	if liquidity.Cmp(balance) == 1 {
		panic(ErrorInsufficientLiquidityBurned)
	}

	amountBip, amountCustom = p.Amounts(liquidity)

	if amountBip.Sign() != 1 || amountCustom.Sign() != 1 {
		panic(ErrorInsufficientLiquidityBurned)
	}

	p.burn(address, liquidity)
	p.update(new(big.Int).Neg(amountBip), new(big.Int).Neg(amountCustom))

	return amountBip, amountCustom
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

	amountBip, amountCustom := p.Amounts(liquidity)

	if amountBip.Sign() != 1 || amountCustom.Sign() != 1 {
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

func (p *Pair) Swap(amountBipIn, amountCustomIn, amountBipOut, amountCustomOut *big.Int) (amount0, amount1 *big.Int, err error) {
	if amountBipOut.Sign() != 1 && amountCustomOut.Sign() != 1 {
		return nil, nil, ErrorInsufficientOutputAmount
	}

	reserve0, reserve1 := p.Reserves()

	if amountBipOut.Cmp(reserve0) == 1 || amountCustomOut.Cmp(reserve1) == 1 {
		return nil, nil, ErrorInsufficientLiquidity
	}

	amount0 = new(big.Int).Sub(amountBipIn, amountBipOut)
	amount1 = new(big.Int).Sub(amountCustomIn, amountCustomOut)

	if amount0.Sign() != 1 && amount1.Sign() != 1 {
		return nil, nil, ErrorInsufficientInputAmount
	}

	balance0Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount0, reserve0), big.NewInt(1000)), new(big.Int).Mul(amountBipIn, big.NewInt(3)))
	balance1Adjusted := new(big.Int).Sub(new(big.Int).Mul(new(big.Int).Add(amount1, reserve1), big.NewInt(1000)), new(big.Int).Mul(amountCustomIn, big.NewInt(3)))

	if new(big.Int).Mul(balance0Adjusted, balance1Adjusted).Cmp(new(big.Int).Mul(new(big.Int).Mul(reserve0, reserve1), big.NewInt(1000000))) == -1 {
		return nil, nil, ErrorK
	}

	p.update(amount0, amount1)

	return amount0, amount1, nil
}

func (p *Pair) mint(address types.Address, value *big.Int) {
	p.Lock()
	defer p.Unlock()

	p.isDirty = true
	p.TotalSupply.Add(p.TotalSupply, value)

	p.muBalances.Lock()
	defer p.muBalances.Unlock()

	balance := p.balances[address]
	if balance == nil {
		p.balances[address] = &Balance{
			Liquidity: big.NewInt(0),
		}
	}

	p.balances[address].isDirty = true
	p.balances[address].Liquidity.Add(p.balances[address].Liquidity, value)
}

func (p *Pair) burn(address types.Address, value *big.Int) {
	p.Lock()
	defer p.Unlock()

	p.isDirty = true
	p.TotalSupply.Sub(p.TotalSupply, value)

	p.muBalances.Lock()
	defer p.muBalances.Unlock()

	p.balances[address].isDirty = true
	p.balances[address].Liquidity.Sub(p.balances[address].Liquidity, value)
}

func (p *Pair) update(amountBip, amountCustom *big.Int) {
	p.Lock()
	defer p.Unlock()

	p.isDirty = true
	p.ReserveBip.Add(p.ReserveBip, amountBip)
	p.ReserveCustom.Add(p.ReserveCustom, amountCustom)
}

func (p *Pair) Amounts(liquidity *big.Int) (amountBip *big.Int, amountCustom *big.Int) {
	p.RLock()
	defer p.RUnlock()
	amountBip = new(big.Int).Div(new(big.Int).Mul(liquidity, p.ReserveBip), p.TotalSupply)
	amountCustom = new(big.Int).Div(new(big.Int).Mul(liquidity, p.ReserveCustom), p.TotalSupply)
	return amountBip, amountCustom
}

func (p *Pair) BoundedAmounts() (amountBip *big.Int, amountCustom *big.Int) {
	boundedSupply := p.Balance(types.Address{})
	return p.Amounts(boundedSupply)
}

func startingSupply(amountBip *big.Int, amountCustom *big.Int) *big.Int {
	mul := new(big.Int).Mul(amountBip, amountCustom)
	sqrt := new(big.Int).Sqrt(mul)
	return new(big.Int).Sub(sqrt, big.NewInt(minimumLiquidity))
}
