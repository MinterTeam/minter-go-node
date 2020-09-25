package coins

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"sort"
	"sync"
)

const (
	mainPrefix = byte('q')
	infoPrefix = byte('i')
)

type Coins struct {
	list  map[types.CoinSymbol]*Model
	dirty map[types.CoinSymbol]struct{}

	bus  *bus.Bus
	iavl tree.MTree

	lock sync.RWMutex
}

func NewCoins(stateBus *bus.Bus, iavl tree.MTree) (*Coins, error) {
	coins := &Coins{bus: stateBus, iavl: iavl, list: map[types.CoinSymbol]*Model{}, dirty: map[types.CoinSymbol]struct{}{}}

	return coins, nil
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

func (c *Coins) get(symbol types.CoinSymbol) *Model {
	if coin := c.getFromMap(symbol); coin != nil {
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

	c.setToMap(symbol, coin)

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

func (c *Coins) Export(state *types.AppState) map[types.CoinSymbol]types.Coin {
	var coins []types.Coin

	c.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix {
			if len(key[1:]) > types.CoinSymbolLength {
				return false
			}

			coinSymbol := types.StrToCoinSymbol(string(key[1:]))
			coin := c.GetCoin(coinSymbol)

			coinModel := types.Coin{
				Name:      coin.Name(),
				Symbol:    coin.Symbol(),
				Volume:    coin.Volume().String(),
				Crr:       uint64(coin.Crr()),
				Reserve:   coin.Reserve().String(),
				MaxSupply: coin.MaxSupply().String(),
				Version:   0,
			}

			coins = append(coins, coinModel)
		}

		return false
	})

	sort.Slice(coins[:], func(i, j int) bool {
		return helpers.StringToBigInt(coins[i].Reserve).Cmp(helpers.StringToBigInt(coins[j].Reserve)) == 1
	})

	coinsMap := make(map[types.CoinSymbol]types.Coin, len(coins))
	coinsMap[types.GetBaseCoin()] = types.Coin{ID: uint64(types.GetBaseCoinID())}

	for i := range coins {
		coins[i].ID = uint64(i + 1)
		coinsMap[coins[i].Symbol] = coins[i]
	}

	state.Coins = coins

	return coinsMap
}

func (c *Coins) getFromMap(symbol types.CoinSymbol) *Model {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[symbol]
}

func (c *Coins) setToMap(symbol types.CoinSymbol, model *Model) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[symbol] = model
}
