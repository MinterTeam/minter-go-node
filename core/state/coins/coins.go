package coins

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sort"
)

const (
	mainPrefix = byte('q')
	infoPrefix = byte('i')
)

const ownerAccountsIndexBucket = "coinOwnerAccountsIndex"
const ownerCandidatesIndexBucket = "coinOwnerCandidatesIndex"
const ownerFrozenFundsIndexBucket = "coinOwnerFrozenFundsIndex"

type Coins struct {
	list  map[types.CoinSymbol]*Model
	dirty map[types.CoinSymbol]struct{}

	bus  *bus.Bus
	iavl tree.Tree
}

func NewCoins(stateBus *bus.Bus, iavl tree.Tree) (*Coins, error) {
	coins := &Coins{bus: stateBus, iavl: iavl, list: map[types.CoinSymbol]*Model{}, dirty: map[types.CoinSymbol]struct{}{}}
	coins.bus.SetCoins(NewBus(coins))

	return coins, nil
}

func (c *Coins) Commit() error {
	coins := c.getOrderedDirtyCoins()
	for _, symbol := range coins {
		coin := c.list[symbol]
		delete(c.dirty, symbol)

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
	if symbol.IsBaseCoin() {
		return true
	}

	return c.get(symbol) != nil
}

func (c *Coins) SubVolume(symbol types.CoinSymbol, amount *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}
	c.get(symbol).SubVolume(amount)
	c.bus.Checker().AddCoinVolume(symbol, big.NewInt(0).Neg(amount))
}

func (c *Coins) AddVolume(symbol types.CoinSymbol, amount *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}
	c.get(symbol).AddVolume(amount)
	c.bus.Checker().AddCoinVolume(symbol, amount)
}

func (c *Coins) SubReserve(symbol types.CoinSymbol, amount *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}
	c.get(symbol).SubReserve(amount)
	c.bus.Checker().AddCoin(types.GetBaseCoin(), amount)
}

func (c *Coins) AddReserve(symbol types.CoinSymbol, amount *big.Int) {
	if symbol.IsBaseCoin() {
		return
	}
	c.get(symbol).AddReserve(amount)
	c.bus.Checker().AddCoin(types.GetBaseCoin(), big.NewInt(0).Neg(amount))
}

func (c *Coins) Create(symbol types.CoinSymbol, name string, volume *big.Int, crr uint, reserve *big.Int, maxSupply *big.Int) {
	coin := &Model{
		CName:      name,
		CCrr:       crr,
		CMaxSupply: maxSupply,
		symbol:     symbol,
		markDirty:  c.markDirty,
		isDirty:    true,
		info: &Info{
			Volume:  big.NewInt(0),
			Reserve: big.NewInt(0),
			isDirty: false,
		},
	}

	c.list[coin.symbol] = coin

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

func (c *Coins) Export(state *types.AppState) {
	// todo: iterate range?
	c.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix {
			coin := c.GetCoin(types.StrToCoinSymbol(string(key[1:])))

			state.Coins = append(state.Coins, types.Coin{
				Name:    coin.Name(),
				Symbol:  coin.Symbol(),
				Volume:  coin.Volume().String(),
				Crr:     coin.Crr(),
				Reserve: coin.Reserve().String(),
			})
		}

		return false
	})
}
