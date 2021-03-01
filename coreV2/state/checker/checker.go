package checker

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"sync"
)

type Checker struct {
	delta       map[types.CoinID]*big.Int
	volumeDelta map[types.CoinID]*big.Int

	lock sync.RWMutex
}

func NewChecker(bus *bus.Bus) *Checker {
	checker := &Checker{
		delta:       map[types.CoinID]*big.Int{},
		volumeDelta: map[types.CoinID]*big.Int{},
	}
	bus.SetChecker(checker)

	return checker
}

func (c *Checker) AddCoin(coin types.CoinID, value *big.Int, msg ...string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cValue, exists := c.delta[coin]

	if !exists {
		cValue = big.NewInt(0)
		c.delta[coin] = cValue
	}

	cValue.Add(cValue, value)
}

func (c *Checker) AddCoinVolume(coin types.CoinID, value *big.Int) {
	c.lock.Lock()
	defer c.lock.Unlock()

	cValue, exists := c.volumeDelta[coin]

	if !exists {
		cValue = big.NewInt(0)
		c.volumeDelta[coin] = cValue
	}

	cValue.Add(cValue, value)
}

// Reset resets checker coin data
func (c *Checker) Reset() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.delta = map[types.CoinID]*big.Int{}
	c.volumeDelta = map[types.CoinID]*big.Int{}
}

// RemoveBaseCoin clears data for base coin for import genesis
func (c *Checker) RemoveBaseCoin() {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.delta, types.GetBaseCoinID())
}

func (c *Checker) deltas() map[types.CoinID]*big.Int {
	return c.delta
}

func (c *Checker) volumeDeltas() map[types.CoinID]*big.Int {
	return c.volumeDelta
}

func (c *Checker) Check() error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	volumeDeltas := c.volumeDeltas()
	for coin, delta := range c.deltas() {
		volume := volumeDeltas[coin]
		if volume == nil {
			volume = big.NewInt(0)
		}

		if delta.Cmp(volume) != 0 {
			return fmt.Errorf("invariants error on coin %s: %s", coin.String(), big.NewInt(0).Sub(volume, delta).String())
		}
	}

	return nil
}
