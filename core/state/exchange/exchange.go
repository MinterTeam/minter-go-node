package exchange

import (
	"errors"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Couple struct {
	XCoin types.CoinID
	YCoin types.CoinID
}

type Liquidity struct {
	XVolume   *big.Int
	YVolume   *big.Int
	providers map[types.Address]*big.Float
}

type Uniswap struct {
	pool           map[Couple]*Liquidity
	providers      map[uint16]types.Address // todo: caching
	lastProviderID uint16
}

func (u *Uniswap) Couple(xCoin types.CoinID, yCoin types.CoinID) (kVolume *big.Int, price *big.Float, err error) {
	panic("implement me")
}

func (u *Uniswap) Couples() ([]*Couple, error) {
	panic("implement me")
}

func NewUniswap() *Uniswap {
	return &Uniswap{pool: map[Couple]*Liquidity{}, providers: map[uint16]types.Address{}}
}

func checkCoins(x types.CoinID, y types.CoinID) (increase bool, err error) {
	if x == y {
		return false, errors.New("equal coins")
	}
	return x < y, nil
}

func (u *Uniswap) Add(provider types.Address, xCoin types.CoinID, xVolume *big.Int, yCoin types.CoinID, yVolume *big.Int) error {
	increase, err := checkCoins(xCoin, yCoin)
	if err != nil {
		return err
	}
	if increase {
		xCoin, yCoin = yCoin, xCoin
		xVolume, yVolume = yVolume, xVolume
	}
	couple := Couple{XCoin: xCoin, YCoin: yCoin}
	liquidity, ok := u.pool[couple]
	if !ok {
		u.pool[couple] = &Liquidity{
			XVolume: xVolume,
			YVolume: yVolume,
			providers: map[types.Address]*big.Float{
				provider: big.NewFloat(100),
			},
		}
		return nil
	}

	currentK := new(big.Int).Add(liquidity.XVolume, liquidity.YVolume)
	kVolume := new(big.Int).Add(xVolume, yVolume)
	totalSum := new(big.Int).Add(currentK, kVolume)
	for _, percent := range liquidity.providers {
		percent = volumeToPercent(totalSum, percentToVolume(currentK, percent))
	}

	percent, ok := liquidity.providers[provider]
	if !ok {
		percent = volumeToPercent(totalSum, kVolume)
		liquidity.providers[provider] = percent
	} else {
		percent = new(big.Float).Add(volumeToPercent(totalSum, kVolume), percent)
	}

	liquidity.XVolume = new(big.Int).Add(liquidity.XVolume, xVolume)
	liquidity.YVolume = new(big.Int).Add(liquidity.YVolume, yVolume)

	return nil
}

func volumeToPercent(total *big.Int, k *big.Int) *big.Float {
	volume := new(big.Float).Quo(new(big.Float).SetInt(total), big.NewFloat(100))
	return new(big.Float).Quo(new(big.Float).SetInt(k), volume)
}
func percentToVolume(total *big.Int, p *big.Float) *big.Int {
	res := new(big.Int)
	volume := new(big.Float).Quo(new(big.Float).SetInt(total), big.NewFloat(100))
	new(big.Float).Mul(volume, p).Int(res)
	return res
}

func (u *Uniswap) Balance(provider types.Address, xCoin types.CoinID, yCoin types.CoinID) (volumes map[types.CoinID]*big.Int, percent *big.Float, err error) {
	increase, err := checkCoins(xCoin, yCoin)
	if err != nil {
		return nil, nil, err
	}
	if increase {
		xCoin, yCoin = yCoin, xCoin
	}
	couple := Couple{XCoin: xCoin, YCoin: yCoin}

	liquidity, ok := u.pool[couple]
	if !ok {
		return nil, nil, errors.New("liquidity not found")
	}
	percent, ok = liquidity.providers[provider]
	if !ok {
		return nil, nil, errors.New("provider balance not found")
	}

	percent = new(big.Float).Set(percent)
	xVolume := percentToVolume(u.pool[couple].XVolume, percent)
	yVolume := percentToVolume(u.pool[couple].YVolume, percent)
	volumes = map[types.CoinID]*big.Int{
		couple.XCoin: xVolume,
		couple.YCoin: yVolume,
	}
	return volumes, percent, nil
}

func (u *Uniswap) Return(provider types.Address, xCoin types.CoinID, yCoin types.CoinID, kVolume *big.Int) (map[types.CoinID]*big.Int, error) {
	increase, err := checkCoins(xCoin, yCoin)
	if err != nil {
		return nil, err
	}
	if increase {
		xCoin, yCoin = yCoin, xCoin
	}
	couple := Couple{XCoin: xCoin, YCoin: yCoin}

	liquidity, ok := u.pool[couple]
	if !ok {
		return nil, errors.New("liquidity not found")
	}
	percent, ok := liquidity.providers[provider]
	if !ok {
		return nil, errors.New("provider balance not found")
	}

	currentK := new(big.Int).Add(liquidity.XVolume, liquidity.YVolume)
	volume := percentToVolume(currentK, percent)
	sub := new(big.Int).Sub(volume, kVolume)
	if sub.Sign() < 0 {
		return nil, errors.New("provider balance less")
	}

	// todo

	if sub.Sign() == 0 {
		delete(liquidity.providers, provider)
	}

	return nil, nil
}

func (u *Uniswap) Exchange(fromCoin types.CoinID, toCoin types.CoinID, volume *big.Int, wantVolume *big.Int) (gotVolume *big.Int, err error) {
	panic("implement me")
}

func (u *Uniswap) Export(state *types.AppState) {
	panic("implement me")
}

func (u *Uniswap) Commit() error {
	panic("implement me")
}

type Exchanger interface {
	Add(provider types.Address, xCoin types.CoinID, xVolume *big.Int, yCoin types.CoinID, yVolume *big.Int) error
	Balance(provider types.Address, xCoin types.CoinID, yCoin types.CoinID) (volumes map[types.CoinID]*big.Int, percent *big.Float, err error)
	Return(provider types.Address, xCoin types.CoinID, yCoin types.CoinID, kVolume *big.Int) (map[types.CoinID]*big.Int, error)
	Exchange(fromCoin types.CoinID, toCoin types.CoinID, volume *big.Int, wantVolume *big.Int) (gotVolume *big.Int, err error)
	Couple(xCoin types.CoinID, yCoin types.CoinID) (kVolume *big.Int, price *big.Float, err error)
	Couples() ([]*Couple, error)
	Export(state *types.AppState)
	Commit() error
}

// Кастомные цепочки? если нет пула для ZERO, при каких условиях стоит ее менять на BIP и искать пары с ним
// сначала нужно проверить с резервом они или без. Если резерв есть, то надо найти максимально короткий путь для обмена.
// Как?

// зачем менять через юнисвоп если можно по-обычному?
// если у монеты предельный объем или количество приближено к резерву

// Что делать с балансом монеты если ее предоставили в пул юнисвоп?
// deliverState.Accounts.SubBalance(sender, ts.Coin, ts.Value)
// но не меняем параметры монеты, такие как Coins.SubVolume и Coins.SubReserve
// поправить валидацию генезиса

// что с комиссиями? будут и куда будут капать?
