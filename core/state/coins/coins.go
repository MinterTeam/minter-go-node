package coins

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
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
	panic("implement me")
}

func (c *Coins) GetCoin(symbol types.CoinSymbol) *Model {
	panic("implement me")
}

func (c *Coins) Exists(symbol types.CoinSymbol) bool {
	panic("implement me")
}

func (c *Coins) SubVolume(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (c *Coins) AddVolume(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (c *Coins) SubReserve(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (c *Coins) AddReserve(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (c *Coins) Sanitize(symbol types.CoinSymbol) {
	panic("implement me")
}

func (c *Coins) Create(symbol types.CoinSymbol, name string, volume *big.Int, crr uint, reserve *big.Int) {
	panic("implement me")
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
