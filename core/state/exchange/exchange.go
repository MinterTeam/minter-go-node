package exchange

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/iavl"
	"log"
	"math/big"
	"sort"
)

type Pair struct {
	XCoin types.CoinID
	YCoin types.CoinID
}

func (p *Pair) Bytes() []byte {
	return append(p.XCoin.Bytes(), p.YCoin.Bytes()...)
}

type Liquidity struct {
	XVolume         *big.Int
	YVolume         *big.Int
	SupplyStakes    *big.Float
	providersStakes map[types.Address]*big.Float
	dirty           bool
}

func newLiquidity(provider types.Address, xVolume *big.Int, yVolume *big.Int) *Liquidity {
	startingStake := startingStake(xVolume, yVolume)
	providers := map[types.Address]*big.Float{provider: new(big.Float).Set(startingStake)}
	return &Liquidity{
		XVolume:         xVolume,
		YVolume:         yVolume,
		SupplyStakes:    startingStake,
		providersStakes: providers,
		dirty:           true,
	}
}

type Provider struct {
	Address types.Address
	Stake   *big.Float
}

func (l *Liquidity) ListStakes() []*Provider {
	providers := make([]*Provider, 0, len(l.providersStakes))
	for address, stake := range l.providersStakes {
		providers = append(providers, &Provider{
			Address: address,
			Stake:   stake,
		})
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Address.Compare(providers[j].Address) == 1
	})
	return providers
}

func (l *Liquidity) stakeToVolumes(stake *big.Float) (xVolume, yVolume *big.Int) {
	xVolume, _ = new(big.Float).Quo(new(big.Float).Mul(new(big.Float).SetInt(l.XVolume), stake), l.SupplyStakes).Int(nil)
	yVolume, _ = new(big.Float).Quo(new(big.Float).Mul(new(big.Float).SetInt(l.YVolume), stake), l.SupplyStakes).Int(nil)
	return xVolume, yVolume
}

type Swap struct {
	pool          map[Pair]*Liquidity
	dirtyPairs    bool
	immutableTree *iavl.ImmutableTree
	loaded        bool
}

func (u *Swap) addPair(pair Pair, liquidity *Liquidity) {
	u.dirtyPairs = true
	u.pool[pair] = liquidity
}

func (u *Swap) Pair(xCoin types.CoinID, yCoin types.CoinID) (xVolume, yVolume *big.Int, err error) {
	reverted, err := checkCoins(xCoin, yCoin)
	if err != nil {
		return nil, nil, err
	}
	if reverted {
		xCoin, yCoin = yCoin, xCoin
		xVolume, yVolume = yVolume, xVolume
	}
	pair := Pair{XCoin: xCoin, YCoin: yCoin}
	liquidity, ok, err := u.liquidity(pair)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, err
	}

	return new(big.Int).Set(liquidity.XVolume), new(big.Int).Set(liquidity.YVolume), nil
}

func (u *Swap) Pairs() (pairs []*Pair) {
	pairs = make([]*Pair, 0, len(u.pool))
	for p := range u.pool {
		pair := p
		pairs = append(pairs, &pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		return bytes.Compare(pairs[i].Bytes(), pairs[j].Bytes()) == 1
	})
	return pairs
}

func NewSwap(db *iavl.ImmutableTree) Exchanger {
	return &Swap{pool: map[Pair]*Liquidity{}, immutableTree: db}
}

func checkCoins(x types.CoinID, y types.CoinID) (reverted bool, err error) {
	if x == y {
		return false, errors.New("equal coins")
	}
	return x > y, nil
}

func startingStake(x *big.Int, y *big.Int) *big.Float {
	return new(big.Float).Sqrt(new(big.Float).SetInt(new(big.Int).Mul(x, y)))
}

func (l *Liquidity) checkStake(xVolume *big.Int, maxYVolume *big.Int, revert bool) (yVolume *big.Int, mintedSupply *big.Float, err error) {
	if revert {
		yVolume, mintedSupply = l.calculateMintingByYVolume(maxYVolume)
		if yVolume.Cmp(xVolume) == 1 {
			return nil, nil, fmt.Errorf("max Y volume %s, calculated Y volume %s", xVolume, yVolume)
		}
		return yVolume, mintedSupply, nil
	}
	yVolume, mintedSupply = l.calculateMintingByXVolume(xVolume)
	if yVolume.Cmp(maxYVolume) == 1 {
		return nil, nil, fmt.Errorf("max Y volume %s, calculated Y volume %s", maxYVolume, yVolume)
	}
	return yVolume, mintedSupply, nil
}

func (l *Liquidity) calculateMintingByXVolume(xVolume *big.Int) (yVolume *big.Int, mintedSupply *big.Float) {
	quo := new(big.Float).Quo(new(big.Float).SetInt(xVolume), new(big.Float).SetInt(l.XVolume))
	yVolume, _ = new(big.Float).Mul(new(big.Float).SetInt(l.YVolume), quo).Int(nil)
	mintedSupply = new(big.Float).Mul(l.SupplyStakes, quo)
	return yVolume, mintedSupply
}

func (l *Liquidity) calculateMintingByYVolume(yVolume *big.Int) (xVolume *big.Int, mintedSupply *big.Float) {
	quo := new(big.Float).Quo(new(big.Float).SetInt(yVolume), new(big.Float).SetInt(l.YVolume))
	xVolume, _ = new(big.Float).Mul(new(big.Float).SetInt(l.XVolume), quo).Int(nil)
	mintedSupply = new(big.Float).Mul(l.SupplyStakes, quo)
	return xVolume, mintedSupply
}

func (l *Liquidity) mint(xVolume *big.Int, maxYVolume *big.Int, revert bool) (*big.Float, error) {
	yVolume, mintedSupply, err := l.checkStake(xVolume, maxYVolume, revert)
	if err != nil {
		return nil, err
	}
	if revert {
		xVolume, yVolume = yVolume, maxYVolume
	}
	l.XVolume = new(big.Int).Add(l.XVolume, xVolume)
	l.YVolume = new(big.Int).Add(l.YVolume, yVolume)
	l.SupplyStakes = new(big.Float).Add(l.SupplyStakes, mintedSupply)
	l.dirty = true
	return mintedSupply, nil
}

func (l *Liquidity) Burn(xVolume, yVolume *big.Int) (burnStake *big.Float) {
	quo := new(big.Float).Quo(new(big.Float).SetInt(xVolume), new(big.Float).SetInt(l.XVolume))
	burnStake = new(big.Float).Mul(l.SupplyStakes, quo)
	l.SupplyStakes = new(big.Float).Sub(l.SupplyStakes, burnStake)
	l.XVolume = new(big.Int).Sub(l.XVolume, xVolume)
	l.YVolume = new(big.Int).Sub(l.YVolume, yVolume)
	l.dirty = true
	return burnStake
}

func (u *Swap) pair(xCoin *types.CoinID, yCoin *types.CoinID, xVolume *big.Int, yVolume *big.Int) (pair Pair, reverted bool, err error) {
	reverted, err = checkCoins(*xCoin, *yCoin)
	if err != nil {
		return Pair{}, false, err
	}
	if reverted {
		*xCoin, *yCoin = *yCoin, *xCoin
		if xVolume != nil && yVolume != nil {
			*xVolume, *yVolume = *yVolume, *xVolume
		}
	}
	pair = Pair{XCoin: *xCoin, YCoin: *yCoin}
	return pair, reverted, nil
}

func (u *Swap) Add(provider types.Address, xCoin types.CoinID, xVolume *big.Int, yCoin types.CoinID, yMaxVolume *big.Int) error {
	yVolume := yMaxVolume
	pair, reverted, err := u.pair(&xCoin, &yCoin, xVolume, yVolume)
	if err != nil {
		return err
	}
	liquidity, ok, err := u.liquidity(pair)
	if err != nil {
		return err
	}
	if !ok {
		u.addPair(pair, newLiquidity(provider, xVolume, yVolume))
		return nil
	}
	mintedSupply, err := liquidity.mint(xVolume, yVolume, reverted)
	if err != nil {
		return err
	}

	liquidity.providersStakes[provider] = new(big.Float).Add(liquidity.providersStakes[provider], mintedSupply)

	return nil
}

func (u *Swap) Balance(provider types.Address, xCoin types.CoinID, yCoin types.CoinID) (xVolume, yVolume *big.Int, providerStake *big.Float, err error) {
	pair, reverted, err := u.pair(&xCoin, &yCoin, nil, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	liquidity, ok, err := u.liquidity(pair)
	if err != nil {
		return nil, nil, nil, err
	}
	if !ok {
		return nil, nil, nil, errors.New("liquidity not found")
	}

	providerStake, ok = liquidity.providersStakes[provider]
	if !ok {
		return nil, nil, nil, errors.New("provider's stake not found")
	}

	xVolume, yVolume = liquidity.stakeToVolumes(providerStake)
	if reverted {
		xVolume, yVolume = yVolume, xVolume
	}

	return xVolume, yVolume, new(big.Float).Set(providerStake), nil
}

func (u *Swap) Remove(provider types.Address, xCoin types.CoinID, yCoin types.CoinID, stake *big.Float) (xVolume, yVolume *big.Int, err error) {
	pair, reverted, err := u.pair(&xCoin, &yCoin, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	liquidity, ok, err := u.liquidity(pair)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, errors.New("liquidity not found")
	}

	providerStake, ok := liquidity.providersStakes[provider]
	if !ok {
		return nil, nil, errors.New("provider's stake not found")
	}

	switch providerStake.Cmp(stake) {
	case -1:
		return nil, nil, errors.New("provider's stake less")
	case 0:
		delete(liquidity.providersStakes, provider)
		liquidity.dirty = true // todo
	case 1:
		liquidity.providersStakes[provider] = providerStake.Sub(providerStake, stake)
		liquidity.dirty = true // todo
	}
	xVolume, yVolume = liquidity.stakeToVolumes(stake)
	liquidity.Burn(xVolume, yVolume)

	if reverted {
		xVolume, yVolume = yVolume, xVolume
	}
	return xVolume, yVolume, nil
}

func (u *Swap) Export(state *types.AppState) {
	panic("implement me")
}

var basePath = []byte("p")

func (u *Swap) Commit(db *iavl.MutableTree) error {
	pairs := u.Pairs()
	if u.dirtyPairs {
		u.dirtyPairs = false
		pairsBytes, err := rlp.EncodeToBytes(pairs)
		if err != nil {
			return err
		}
		db.Set(basePath, pairsBytes)
	}
	for _, pair := range pairs {
		liquidity, _, err := u.liquidity(*pair)
		if err != nil {
			return err
		}
		if !liquidity.dirty {
			continue
		}

		liquidity.dirty = false

		pairPath := append(basePath, pair.Bytes()...)
		stakesPath := append(pairPath, []byte("s")...)

		if liquidity.SupplyStakes.Sign() != 1 || liquidity.YVolume.Sign() != 1 || liquidity.XVolume.Sign() != 1 {
			db.Remove(pairPath)
			db.Remove(stakesPath)
			continue
		}
		liquidityBytes, err := rlp.EncodeToBytes(liquidity)
		if err != nil {
			return err
		}
		db.Set(pairPath, liquidityBytes)

		pairStakes, err := rlp.EncodeToBytes(liquidity.ListStakes())
		if err != nil {
			return err
		}
		db.Set(stakesPath, pairStakes)
	}
	return nil
}

func (u *Swap) liquidity(pair Pair) (liquidity *Liquidity, ok bool, err error) {
	liquidity, ok = u.pool[pair]
	if ok {
		return liquidity, ok, nil
	}
	if u.loaded {
		return nil, false, nil
	}
	u.loaded = true
	pairPath := append(basePath, pair.Bytes()...)
	_, pairBytes := u.immutableTree.Get(pairPath)
	if len(pairBytes) == 0 {
		return nil, false, nil
	}
	liquidity = new(Liquidity)
	err = rlp.DecodeBytes(pairBytes, liquidity)
	if err != nil {
		return nil, false, err
	}
	log.Printf("%#v", liquidity)
	stakesPath := append(pairPath, []byte("s")...)
	_, pairStakesBytes := u.immutableTree.Get(stakesPath)
	if len(pairStakesBytes) == 0 {
		return nil, false, nil
	}
	var pearStakes []*Provider
	err = rlp.DecodeBytes(pairStakesBytes, &pearStakes)
	if err != nil {
		return nil, false, err
	}
	liquidity.providersStakes = map[types.Address]*big.Float{}
	for _, provider := range pearStakes {
		liquidity.providersStakes[provider.Address] = provider.Stake
		log.Printf("%#v", provider.Stake)
	}
	u.pool[pair] = liquidity
	return liquidity, true, nil
}

type Exchanger interface {
	Add(provider types.Address, xCoin types.CoinID, xVolume *big.Int, yCoin types.CoinID, yMaxVolume *big.Int) error
	Balance(provider types.Address, xCoin types.CoinID, yCoin types.CoinID) (xVolume, yVolume *big.Int, stake *big.Float, err error)
	Remove(provider types.Address, xCoin types.CoinID, yCoin types.CoinID, stake *big.Float) (xVolume, yVolume *big.Int, err error)
	// todo: add
	// SellAll
	// Sell
	// BuyAll
	// Buy

	// fromCoin...toCoin []types.CoinID,
	// Exchange(path []types.CoinID, fromVolume *big.Int, minToVolume *big.Int) (gotVolume *big.Int, err error)

	Pair(xCoin types.CoinID, yCoin types.CoinID) (xVolume, yVolume *big.Int, err error)
	Pairs() []*Pair
	Export(state *types.AppState)
	Commit(db *iavl.MutableTree) error
}
