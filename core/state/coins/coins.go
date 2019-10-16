package coins

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/xujiajun/nutsdb"
	"math/big"
	"sort"
)

const (
	mainPrefix = byte('q')
	infoPrefix = byte('i')
)

const ownerIndexBucket = "coinOwnerIndex"

type Coins struct {
	list  map[types.CoinSymbol]*Model
	dirty map[types.CoinSymbol]struct{}

	bus  *bus.Bus
	iavl tree.Tree
	db   *nutsdb.DB
}

func NewCoins(stateBus *bus.Bus, iavl tree.Tree, db *nutsdb.DB) (*Coins, error) {
	coins := &Coins{bus: stateBus, iavl: iavl, db: db}
	coins.bus.SetCoins(NewBus(coins))

	return coins, nil
}

func (c *Coins) Commit() error {
	coins := c.getOrderedDirtyCoins()
	for _, symbol := range coins {
		coin := c.list[symbol]
		delete(c.dirty, symbol)

		if coin.isDeleted {
			c.iavl.Remove(append([]byte{mainPrefix}, symbol[:]...))

			path := append([]byte{mainPrefix}, symbol[:]...)
			path = append(path, infoPrefix)
			c.iavl.Remove(path)
			continue
		}

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
	coin := c.GetCoin(symbol)
	if coin.isDeleted {
		return
	}
	coin.isDeleted = true

	accounts, candidates, frozenfunds := c.getOwners(symbol)
	for _, address := range accounts {
		c.bus.Accounts().DeleteCoin(address, symbol)
	}

	for _, pubkey := range candidates {
		c.bus.Candidates().DeleteCoin(pubkey, symbol)
	}

	for _, height := range frozenfunds {
		c.bus.FrozenFunds().DeleteCoin(height, symbol)
	}
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

func (c *Coins) AddOwnerAddress(symbol types.CoinSymbol, address types.Address) {
	err := c.db.Update(func(tx *nutsdb.Tx) error {
		return tx.SAdd(ownerIndexBucket, symbol.Bytes(), address.Bytes())
	})
	if err != nil {
		panic(err)
	}
}

func (c *Coins) RemoveOwnerAddress(symbol types.CoinSymbol, address types.Address) {
	err := c.db.Update(func(tx *nutsdb.Tx) error {
		return tx.SRem(ownerIndexBucket, symbol.Bytes(), address.Bytes())
	})
	if err != nil {
		panic(err)
	}
}

func (c *Coins) getOwners(symbol types.CoinSymbol) ([]types.Address, []types.Pubkey, []uint64) {
	var owners []types.Address
	err := c.db.View(func(tx *nutsdb.Tx) error {
		items, err := tx.SMembers(ownerIndexBucket, symbol.Bytes())
		if err != nil {
			return err
		}

		for _, item := range items {
			owners = append(owners, types.BytesToAddress(item))
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	return owners, nil, nil
}
