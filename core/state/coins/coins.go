package coins

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
)

const mainPrefix = byte('q')
const infoPrefix = byte('i')

type Coins struct {
	list  map[types.CoinSymbol]*Model
	dirty map[types.CoinSymbol]struct{}

	iavl tree.Tree
}

func NewCoins(iavl tree.Tree) (*Coins, error) {
	return &Coins{iavl: iavl}, nil
}

func (c *Coins) Commit() error {
	coins := c.getOrderedDirtyCoins()
	for _, symbol := range coins {
		coin := c.list[symbol]

		if coin.IsDirty() {
			data, err := rlp.EncodeToBytes(coin)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", symbol[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, symbol[:]...)
			c.iavl.Set(path, data)
			coin.isDirty = false
		}

		if coin.IsInfoDirty() {
			data, err := rlp.EncodeToBytes(coin.info)
			if err != nil {
				return fmt.Errorf("can't encode object at %x: %v", symbol[:], err)
			}

			path := []byte{mainPrefix}
			path = append(path, symbol[:]...)
			path = append(path, infoPrefix)
			c.iavl.Set(path, data)
			coin.info.isDirty = false
		}

	}

	return nil
}

func (c *Coins) GetCoin(symbol types.CoinSymbol) *Model {
	return c.get(symbol)
}

func (c *Coins) Exists(symbol types.CoinSymbol) bool {
	return c.get(symbol) != nil
}

func (c *Coins) SubVolume(symbol types.CoinSymbol, amount *big.Int) {
	c.get(symbol).SubVolume(amount)
}

func (c *Coins) AddVolume(symbol types.CoinSymbol, amount *big.Int) {
	c.get(symbol).AddVolume(amount)
}

func (c *Coins) SubReserve(symbol types.CoinSymbol, amount *big.Int) {
	c.get(symbol).SubReserve(amount)
}

func (c *Coins) AddReserve(symbol types.CoinSymbol, amount *big.Int) {
	c.get(symbol).AddReserve(amount)
}

func (c *Coins) Sanitize(symbol types.CoinSymbol) {
	if symbol.IsBaseCoin() {
		return
	}

	coin := c.get(symbol)
	if coin.IsToDelete() {
		c.delete(coin.symbol)
	}
}

func (c *Coins) Create(symbol types.CoinSymbol, name string, volume *big.Int, crr uint, reserve *big.Int) {
	coin := Model{
		CName:     name,
		CCrr:      crr,
		symbol:    symbol,
		markDirty: c.markDirty,
		isDirty:   true,
	}

	coin.SetReserve(reserve)
	coin.SetVolume(volume)
	c.markDirty(symbol)
}

func (c *Coins) get(symbol types.CoinSymbol) *Model {
	if coin := c.list[symbol]; coin != nil {
		return coin
	}

	path := []byte{mainPrefix}
	path = append(path, symbol[:]...)
	_, enc := c.iavl.Get(path)
	if len(enc) == 0 {
		return nil
	}

	coin := &Model{}
	if err := rlp.DecodeBytes(enc, coin); err != nil {
		panic(fmt.Sprintf("failed to decode coin at %s: %s", symbol.String(), err))
		return nil
	}

	coin.symbol = symbol
	coin.markDirty = c.markDirty

	// load info
	path = []byte{mainPrefix}
	path = append(path, symbol[:]...)
	path = append(path, infoPrefix)
	_, enc = c.iavl.Get(path)
	if len(enc) != 0 {
		var info Info
		if err := rlp.DecodeBytes(enc, &info); err != nil {
			panic(fmt.Sprintf("failed to decode coin info %s: %s", symbol.String(), err))
		}

		coin.info = &info
	}

	c.list[symbol] = coin

	return coin
}

func (c *Coins) markDirty(symbol types.CoinSymbol) {
	c.dirty[symbol] = struct{}{}
}

func (c *Coins) delete(symbol types.CoinSymbol) {
	// todo: delete coin
}

func (c *Coins) getOrderedDirtyCoins() []types.CoinSymbol {
	keys := make([]types.CoinSymbol, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}
