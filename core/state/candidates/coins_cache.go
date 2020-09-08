package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type coinsCache struct {
	list map[types.CoinID]*coinsCacheItem
}

func newCoinsCache() *coinsCache {
	return &coinsCache{list: map[types.CoinID]*coinsCacheItem{}}
}

type coinsCacheItem struct {
	saleReturn  *big.Int
	totalAmount *big.Int
}

func (c *coinsCache) Exists(id types.CoinID) bool {
	if c == nil {
		return false
	}

	_, exists := c.list[id]

	return exists
}

func (c *coinsCache) Get(id types.CoinID) (saleReturn *big.Int, totalAmount *big.Int) {
	return big.NewInt(0).Set(c.list[id].totalAmount), big.NewInt(0).Set(c.list[id].totalAmount)
}

func (c *coinsCache) Set(id types.CoinID, saleReturn *big.Int, totalAmount *big.Int) {
	if c == nil {
		return
	}

	if c.list[id] == nil {
		c.list[id] = &coinsCacheItem{}
	}

	c.list[id].totalAmount = totalAmount
	c.list[id].saleReturn = saleReturn
}
