package coins

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sort"
	"strconv"
	"sync"
)

func MaxCoinSupply() *big.Int {
	return big.NewInt(0).Exp(big.NewInt(10), big.NewInt(15+18), nil)
}

// Deprecated
type modelV1 struct {
	CName      string
	CCrr       uint32
	CMaxSupply *big.Int
	CVersion   types.CoinVersion
	CSymbol    types.CoinSymbol

	id         types.CoinID
	info       *Info
	symbolInfo *SymbolInfo

	markDirty func(symbol types.CoinID)
	lock      sync.RWMutex

	isDirty   bool
	isCreated bool
}

// Deprecated
func (c *Coins) ExportV1(state *types.AppState, subValues map[types.CoinID]*big.Int, owners map[types.CoinID]*accounts.MaxCoinVolume) (types.CoinID, *big.Int) {
	totalSubReserve := big.NewInt(0)
	c.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		if len(key) > 5 {
			return false
		}

		coinID := types.BytesToCoinID(key[1:])
		coinV1 := c.getV1(coinID)

		coin := &Model{
			CName:      coinV1.CName,
			CCrr:       coinV1.CCrr,
			CMaxSupply: coinV1.CMaxSupply,
			CVersion:   coinV1.CVersion,
			CSymbol:    coinV1.CSymbol,
			Mintable:   false,
			Burnable:   false,
			id:         coinID,
			info:       coinV1.info,
			symbolInfo: coinV1.symbolInfo,
			markDirty:  nil,
			lock:       sync.RWMutex{},
			isDirty:    false,
			isCreated:  false,
		}

		volume := coin.Volume()
		reserve := coin.Reserve()

		subValue, has := subValues[coinID]
		if has {
			// if coinID != types.GetBaseCoinID() {
			subReserve := formula.CalculateSaleReturn(volume, reserve, coin.CCrr, subValue)
			reserve.Sub(reserve, subReserve)
			totalSubReserve.Add(totalSubReserve, subReserve)
			// } else {
			// 	totalSubReserve.Add(totalSubReserve, subValue)
			// }
			volume.Sub(volume, subValue)
		}

		symbol := coin.Symbol()
		strSymbol := symbol.String()
		if _, err := strconv.Atoi(strSymbol); err == nil || coinID != 0 && strSymbol == types.GetBaseCoin().String() {
			symbol = types.StrToCoinSymbol(strSymbol + "A")
		}

		var owner *types.Address
		info := c.getSymbolInfo(coin.Symbol())
		if info != nil {
			if coinID != 969 && coinID != 905 {
				owner = info.OwnerAddress()
			}
		} else if v, ok := owners[coinID]; ok {
			mul := big.NewInt(0).Mul(v.Volume, big.NewInt(100))
			div := mul.Div(mul, volume)
			if div.Cmp(big.NewInt(90)) != -1 {
				// log.Println("fix owner of coin", symbol, v.Owner.String())
				owner = &v.Owner
			}
		}

		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(coin.ID()),
			Name:         coin.Name(),
			Symbol:       symbol,
			Volume:       volume.String(),
			Crr:          uint64(coin.Crr()),
			Reserve:      reserve.String(),
			MaxSupply:    coin.MaxSupply().String(),
			Version:      uint64(coin.Version()),
			OwnerAddress: owner,
			Mintable:     false,
			Burnable:     false,
		})

		return false
	})

	sort.Slice(state.Coins[:], func(i, j int) bool {
		return state.Coins[i].ID < state.Coins[j].ID
	})

	usdcID := state.Coins[len(state.Coins)-1].ID + 1

	bridge := types.HexToAddress("Mxffffffffffffffffffffffffffffffffffffffff")
	state.Coins = append(state.Coins, types.Coin{
		ID:           usdcID,
		Name:         "USDC",
		Symbol:       types.StrToCoinSymbol("MUSDC"),
		Volume:       helpers.BipToPip(big.NewInt(1000000000)).String(),
		Crr:          0,
		Reserve:      "0",
		MaxSupply:    MaxCoinSupply().String(),
		Version:      0,
		OwnerAddress: &bridge,
		Mintable:     true,
		Burnable:     true,
	})

	return types.CoinID(usdcID), totalSubReserve
}

// Deprecated
func (c *Coins) getV1(id types.CoinID) *modelV1 {
	if id.IsBaseCoin() {
		return &modelV1{
			id:         types.GetBaseCoinID(),
			CSymbol:    types.GetBaseCoin(),
			CMaxSupply: helpers.BipToPip(big.NewInt(10000000000)),
			info: &Info{
				Volume:  big.NewInt(0),
				Reserve: big.NewInt(0),
			},
		}
	}

	// if coin := c.getFromMap(id); coin != nil {
	// 	return coin
	// }

	_, enc := c.immutableTree().Get(getCoinPath(id))
	if len(enc) == 0 {
		return nil
	}

	coin := &modelV1{}
	if err := rlp.DecodeBytes(enc, coin); err != nil {
		panic(fmt.Sprintf("failed to decode coin at %d: %s", id, err))
	}

	coin.lock.Lock()
	coin.id = id
	coin.markDirty = c.markDirty
	coin.lock.Unlock()

	// load info
	_, enc = c.immutableTree().Get(getCoinInfoPath(id))
	if len(enc) != 0 {
		var info Info
		if err := rlp.DecodeBytes(enc, &info); err != nil {
			panic(fmt.Sprintf("failed to decode coin info %d: %s", id, err))
		}

		coin.lock.Lock()
		coin.info = &info
		coin.lock.Unlock()
	}

	// c.setToMap(id, coin)

	return coin
}

// Deprecated
func (b *Bus) GetCoinV1(id types.CoinID) *bus.Coin {
	coin := b.coins.GetCoinV1(id)
	if coin == nil {
		return nil
	}

	return &bus.Coin{
		ID:      coin.id,
		Name:    coin.Name(),
		Crr:     coin.Crr(),
		Symbol:  coin.Symbol(),
		Volume:  coin.Volume(),
		Reserve: coin.Reserve(),
		Version: coin.Version(),
	}
}

// Deprecated
func (c *Coins) GetCoinV1(id types.CoinID) *Model {
	v1 := c.getV1(id)
	return &Model{
		CName:      v1.CName,
		CCrr:       v1.CCrr,
		CMaxSupply: v1.CMaxSupply,
		CVersion:   v1.CVersion,
		CSymbol:    v1.CSymbol,
		Mintable:   false,
		Burnable:   false,
		id:         v1.id,
		info:       v1.info,
		symbolInfo: v1.symbolInfo,
		markDirty:  nil,
		lock:       sync.RWMutex{},
		isDirty:    false,
		isCreated:  false,
	}
}
