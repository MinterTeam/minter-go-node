package checker

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Checker struct {
	delta       map[types.CoinSymbol]*big.Int
	volumeDelta map[types.CoinSymbol]*big.Int
}

func NewChecker(bus *bus.Bus) *Checker {
	checker := &Checker{
		delta:       map[types.CoinSymbol]*big.Int{},
		volumeDelta: map[types.CoinSymbol]*big.Int{},
	}
	bus.SetChecker(checker)

	return checker
}

func (c *Checker) AddCoin(coin types.CoinSymbol, value *big.Int) {
	cValue, exists := c.delta[coin]

	if !exists {
		cValue = big.NewInt(0)
		c.delta[coin] = cValue
	}

	cValue.Add(cValue, value)
}

func (c *Checker) AddCoinVolume(coin types.CoinSymbol, value *big.Int) {
	cValue, exists := c.volumeDelta[coin]

	if !exists {
		cValue = big.NewInt(0)
		c.volumeDelta[coin] = cValue
	}

	cValue.Add(cValue, value)
}

func (c *Checker) Reset() {
	c.delta = map[types.CoinSymbol]*big.Int{}
	c.volumeDelta = map[types.CoinSymbol]*big.Int{}
}

func (c *Checker) Deltas() map[types.CoinSymbol]*big.Int {
	return c.delta
}

func (c *Checker) VolumeDeltas() map[types.CoinSymbol]*big.Int {
	return c.volumeDelta
}
