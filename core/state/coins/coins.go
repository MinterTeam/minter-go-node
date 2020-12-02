package coins

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/iavl"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
)

const (
	mainPrefix   = byte('q')
	infoPrefix   = byte('i')
	symbolPrefix = byte('s')

	BaseVersion types.CoinVersion = 0
)

type RCoins interface {
	Export(state *types.AppState)
	Exists(id types.CoinID) bool
	ExistsBySymbol(symbol types.CoinSymbol) bool
	SubReserve(symbol types.CoinID, amount *big.Int)
	GetCoin(id types.CoinID) *Model
	GetCoinBySymbol(symbol types.CoinSymbol, version types.CoinVersion) *Model
	GetSymbolInfo(symbol types.CoinSymbol) *SymbolInfo
}

// Coins represents coins state in blockchain.
//
// When a coin is created with a CreateCoinTx transaction, such model is created:
// Model {
//  id
//  info
//  symbolInfo
//
//  Name
//  Crr
//  MaxSupply
//  Version - is the version of the coin
//  Symbol - is the base symbol of the coin
// }
//
// Also, SymbolInfo is created:
// SymbolInfo {
//  OwnerAddress
// }
//
// It is a structure that retains the owner of the ticker (not the coin).
// coin.symbolInfo is saved in the symbolsInfoList map, in which the key is the symbol,
// and the value is the owner of the coin, and upon commit is written to the db by this key:
// mainPrefix + symbolPrefix + symbol + infoPrefix.
//
// Also, you need to save all coins for a particular ticker. That is, you need to associate
// the ticker with all the coins that refer to it. For this, there is a map symbolsList,
// in which the key is the ticker, and the value is an array of ids of coins that
// belong to this ticker (just with a different version). When you commit, this array is
// saved to db by this key: mainPrefix + symbolPrefix + symbol.
//
// The coin model is saved at: mainPrefix + id.
//
// When a coin is re-created with a RecreateCoinTx transaction, the state retrieves an array of
// coins that refer to this ticker (getBySymbol). Finds the current current version there, changes
// it to the new version. And the new coin is assigned version 0. The new coin is also added to symbolsList [ticker].
//
// When changing the owner with a ChangeOwnerTx transaction, the state gets the current owner
// getSymbolInfo (ticker) and changes the owner there and saves it back.
type Coins struct {
	list            map[types.CoinID]*Model
	dirty           map[types.CoinID]struct{}
	symbolsList     map[types.CoinSymbol][]types.CoinID
	symbolsInfoList map[types.CoinSymbol]*SymbolInfo

	bus *bus.Bus
	db  atomic.Value

	lock sync.RWMutex
}

func NewCoins(stateBus *bus.Bus, db *iavl.ImmutableTree) *Coins {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	coins := &Coins{
		bus:             stateBus,
		db:              immutableTree,
		list:            map[types.CoinID]*Model{},
		dirty:           map[types.CoinID]struct{}{},
		symbolsList:     map[types.CoinSymbol][]types.CoinID{},
		symbolsInfoList: map[types.CoinSymbol]*SymbolInfo{},
	}
	coins.bus.SetCoins(NewBus(coins))

	return coins
}

func (c *Coins) immutableTree() *iavl.ImmutableTree {
	db := c.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (c *Coins) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	c.db.Store(immutableTree)
}

func (c *Coins) Commit(db *iavl.MutableTree) error {
	coins := c.getOrderedDirtyCoins()
	for _, id := range coins {
		coin := c.getFromMap(id)
		c.lock.Lock()
		delete(c.dirty, id)
		c.lock.Unlock()

		if coin.IsCreated() {
			ids := c.getBySymbol(coin.Symbol())
			data, err := rlp.EncodeToBytes(ids)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			db.Set(getSymbolCoinsPath(coin.Symbol()), data)
			coin.isCreated = false
		}

		if coin.IsDirty() {
			data, err := rlp.EncodeToBytes(coin)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			db.Set(getCoinPath(id), data)
			coin.isDirty = false
		}

		if coin.IsInfoDirty() {
			data, err := rlp.EncodeToBytes(coin.info)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			db.Set(getCoinInfoPath(id), data)
			coin.info.isDirty = false
		}

		if coin.IsSymbolInfoDirty() {
			data, err := rlp.EncodeToBytes(coin.symbolInfo)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", id, err)
			}

			db.Set(getSymbolInfoPath(coin.Symbol()), data)
			coin.symbolInfo.isDirty = false
		}
	}

	return nil
}

func (c *Coins) GetCoin(id types.CoinID) *Model {
	return c.get(id)
}

func (c *Coins) GetSymbolInfo(symbol types.CoinSymbol) *SymbolInfo {
	return c.getSymbolInfo(symbol)
}

func (c *Coins) Exists(id types.CoinID) bool {
	if id.IsBaseCoin() {
		return true
	}

	return c.get(id) != nil
}

func (c *Coins) ExistsBySymbol(symbol types.CoinSymbol) bool {
	if symbol.IsBaseCoin() {
		return true
	}

	return c.getBySymbol(symbol) != nil
}

func (c *Coins) GetCoinBySymbol(symbol types.CoinSymbol, version types.CoinVersion) *Model {
	if symbol.IsBaseCoin() {
		return c.get(types.GetBaseCoinID())
	}

	coins := c.getBySymbol(symbol)
	if len(coins) == 0 {
		return nil
	}

	for _, coinID := range coins {
		coin := c.get(coinID)
		if coin.Version() == version {
			return coin
		}
	}

	return nil
}

func (c *Coins) SubVolume(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).SubVolume(amount)
	c.bus.Checker().AddCoinVolume(id, big.NewInt(0).Neg(amount))
}

func (c *Coins) AddVolume(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).AddVolume(amount)
	c.bus.Checker().AddCoinVolume(id, amount)
}

func (c *Coins) SubReserve(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).SubReserve(amount)
	c.bus.Checker().AddCoin(types.GetBaseCoinID(), big.NewInt(0).Neg(amount))
}

func (c *Coins) AddReserve(id types.CoinID, amount *big.Int) {
	if id.IsBaseCoin() {
		return
	}

	c.get(id).AddReserve(amount)
	c.bus.Checker().AddCoin(types.GetBaseCoinID(), amount)
}

func (c *Coins) Create(id types.CoinID, symbol types.CoinSymbol, name string,
	volume *big.Int, crr uint32, reserve *big.Int, maxSupply *big.Int, owner *types.Address,
) {
	coin := &Model{
		CName:      name,
		CCrr:       crr,
		CMaxSupply: maxSupply,
		CSymbol:    symbol,
		id:         id,
		markDirty:  c.markDirty,
		isDirty:    true,
		isCreated:  true,
		info: &Info{
			Volume:  volume,
			Reserve: reserve,
			isDirty: true,
		},
	}

	if owner != nil {
		coin.symbolInfo = &SymbolInfo{
			COwnerAddress: owner,
			isDirty:       true,
		}

		c.setSymbolInfoToMap(coin.symbolInfo, coin.Symbol())
	}

	ids := c.getBySymbol(coin.Symbol())
	ids = append(ids, coin.ID())

	c.setSymbolToMap(ids, coin.Symbol())
	c.setToMap(coin.ID(), coin)

	if reserve != nil {
		c.bus.Checker().AddCoin(types.GetBaseCoinID(), reserve)
	}

	c.bus.Checker().AddCoinVolume(coin.id, volume)

	c.markDirty(coin.id)
}

func (c *Coins) Recreate(newID types.CoinID, name string, symbol types.CoinSymbol,
	volume *big.Int, crr uint32, reserve *big.Int, maxSupply *big.Int,
) {
	recreateCoin := c.GetCoinBySymbol(symbol, BaseVersion)
	if recreateCoin == nil {
		panic("coin to recreate does not exists")
	}

	// update version for recreating coin
	symbolCoins := c.getBySymbol(symbol)

	lastVersion := uint16(0)
	for _, id := range symbolCoins {
		coin := c.get(id)
		if coin.Version() > lastVersion {
			lastVersion = coin.Version()
		}
	}

	recreateCoin.CVersion = lastVersion + 1
	recreateCoin.isDirty = true

	c.setToMap(recreateCoin.id, recreateCoin)
	c.markDirty(recreateCoin.id)

	c.Create(newID, recreateCoin.Symbol(), name, volume, crr, reserve, maxSupply, nil)
}

func (c *Coins) ChangeOwner(symbol types.CoinSymbol, owner types.Address) {
	info := c.getSymbolInfo(symbol)
	info.setOwnerAddress(owner)

	coin := c.GetCoinBySymbol(symbol, BaseVersion)
	coin.symbolInfo = info

	c.setSymbolInfoToMap(coin.symbolInfo, coin.Symbol())
	c.setToMap(coin.ID(), coin)
	c.markDirty(coin.ID())
}

func (c *Coins) get(id types.CoinID) *Model {
	if id.IsBaseCoin() {
		return &Model{
			id:         types.GetBaseCoinID(),
			CSymbol:    types.GetBaseCoin(),
			CMaxSupply: helpers.BipToPip(big.NewInt(10000000000)),
			info: &Info{
				Volume:  big.NewInt(0),
				Reserve: big.NewInt(0),
			},
		}
	}

	if coin := c.getFromMap(id); coin != nil {
		return coin
	}

	_, enc := c.immutableTree().Get(getCoinPath(id))
	if len(enc) == 0 {
		return nil
	}

	coin := &Model{}
	if err := rlp.DecodeBytes(enc, coin); err != nil {
		panic(fmt.Sprintf("failed to decode coin at %d: %s", id, err))
	}

	coin.id = id
	coin.markDirty = c.markDirty

	// load info
	_, enc = c.immutableTree().Get(getCoinInfoPath(id))
	if len(enc) != 0 {
		var info Info
		if err := rlp.DecodeBytes(enc, &info); err != nil {
			panic(fmt.Sprintf("failed to decode coin info %d: %s", id, err))
		}

		coin.info = &info
	}

	c.setToMap(id, coin)

	return coin
}

func (c *Coins) getSymbolInfo(symbol types.CoinSymbol) *SymbolInfo {
	if info, ok := c.getSymbolInfoFromMap(symbol); ok {
		return info
	}

	info := &SymbolInfo{}

	_, enc := c.immutableTree().Get(getSymbolInfoPath(symbol))
	if len(enc) == 0 {
		return nil
	}

	if err := rlp.DecodeBytes(enc, info); err != nil {
		panic(fmt.Sprintf("failed to decode coin symbol %s: %s", symbol.String(), err))
	}

	c.setSymbolInfoToMap(info, symbol)

	return info
}

func (c *Coins) getBySymbol(symbol types.CoinSymbol) []types.CoinID {
	if coins, ok := c.getSymbolFromMap(symbol); ok {
		return coins
	}

	var coins []types.CoinID

	_, enc := c.immutableTree().Get(getSymbolCoinsPath(symbol))
	if len(enc) == 0 {
		return coins
	}

	if err := rlp.DecodeBytes(enc, &coins); err != nil {
		panic(fmt.Sprintf("failed to decode coins by symbol %s: %s", symbol, err))
	}

	c.setSymbolToMap(coins, symbol)

	return coins
}

func (c *Coins) markDirty(id types.CoinID) {
	c.dirty[id] = struct{}{}
}

func (c *Coins) getOrderedDirtyCoins() []types.CoinID {
	keys := make([]types.CoinID, 0, len(c.dirty))
	for k := range c.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	return keys
}

func (c *Coins) Export(state *types.AppState) {
	c.immutableTree().IterateRange([]byte{mainPrefix}, []byte{mainPrefix + 1}, true, func(key []byte, value []byte) bool {
		if len(key) > 5 {
			return false
		}

		coinID := types.BytesToCoinID(key[1:])
		coin := c.get(coinID)

		var owner *types.Address
		info := c.getSymbolInfo(coin.Symbol())
		if info != nil {
			owner = info.OwnerAddress()
		}

		var reservePointer *string
		if coin.HasReserve() {
			reserve := coin.Reserve().String()
			reservePointer = &reserve
		}
		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(coin.ID()),
			Name:         coin.Name(),
			Symbol:       coin.Symbol(),
			Volume:       coin.Volume().String(),
			Crr:          uint64(coin.Crr()),
			Reserve:      reservePointer,
			MaxSupply:    coin.MaxSupply().String(),
			Version:      uint64(coin.Version()),
			OwnerAddress: owner,
		})

		return false
	})

	sort.Slice(state.Coins[:], func(i, j int) bool {
		return state.Coins[i].ID < state.Coins[j].ID
	})
}

func (c *Coins) getFromMap(id types.CoinID) *Model {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.list[id]
}

func (c *Coins) setToMap(id types.CoinID, model *Model) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.list[id] = model
}

func (c *Coins) getSymbolInfoFromMap(symbol types.CoinSymbol) (*SymbolInfo, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	info, ok := c.symbolsInfoList[symbol]
	return info, ok
}

func (c *Coins) setSymbolInfoToMap(info *SymbolInfo, symbol types.CoinSymbol) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.symbolsInfoList[symbol] = info
}

func (c *Coins) getSymbolFromMap(symbol types.CoinSymbol) ([]types.CoinID, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	coins, ok := c.symbolsList[symbol]
	return coins, ok
}

func (c *Coins) setSymbolToMap(coins []types.CoinID, symbol types.CoinSymbol) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.symbolsList[symbol] = coins
}

func getSymbolCoinsPath(symbol types.CoinSymbol) []byte {
	path := append([]byte{mainPrefix}, []byte{symbolPrefix}...)
	return append(path, symbol.Bytes()...)
}

func getSymbolInfoPath(symbol types.CoinSymbol) []byte {
	path := append([]byte{mainPrefix}, []byte{symbolPrefix}...)
	path = append(path, symbol.Bytes()...)
	return append(path, []byte{infoPrefix}...)
}

func getCoinPath(id types.CoinID) []byte {
	return append([]byte{mainPrefix}, id.Bytes()...)
}

func getCoinInfoPath(id types.CoinID) []byte {
	return append(getCoinPath(id), infoPrefix)
}
