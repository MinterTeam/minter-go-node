package app

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/cosmos/iavl"
	"math/big"
	"sync/atomic"
)

const mainPrefix = 'd'

type RApp interface {
	Export(state *types.AppState)
	GetMaxGas() uint64
	GetTotalSlashed() *big.Int
	GetCoinsCount() uint32
	GetNextCoinID() types.CoinID
}

type App struct {
	model   *Model
	isDirty bool

	db atomic.Value

	bus *bus.Bus
}

func NewApp(stateBus *bus.Bus, db *iavl.ImmutableTree) *App {
	immutableTree := atomic.Value{}
	if db != nil {
		immutableTree.Store(db)
	}
	app := &App{bus: stateBus, db: immutableTree}
	app.bus.SetApp(NewBus(app))

	return app
}

func (a *App) immutableTree() *iavl.ImmutableTree {
	db := a.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (a *App) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	a.db.Store(immutableTree)
}

func (a *App) Commit(db *iavl.MutableTree) error {
	if !a.isDirty {
		return nil
	}

	a.isDirty = false

	data, err := rlp.EncodeToBytes(a.model)
	if err != nil {
		return fmt.Errorf("can't encode legacyApp model: %s", err)
	}

	path := []byte{mainPrefix}
	db.Set(path, data)

	return nil
}

func (a *App) GetMaxGas() uint64 {
	model := a.getOrNew()

	return model.getMaxGas()
}

func (a *App) SetMaxGas(gas uint64) {
	model := a.getOrNew()
	model.setMaxGas(gas)
}

func (a *App) GetTotalSlashed() *big.Int {
	model := a.getOrNew()

	return model.getTotalSlashed()
}

func (a *App) AddTotalSlashed(amount *big.Int) {
	if amount.Cmp(big.NewInt(0)) == 0 {
		return
	}

	model := a.getOrNew()
	model.setTotalSlashed(big.NewInt(0).Add(model.getTotalSlashed(), amount))
	a.bus.Checker().AddCoin(types.GetBaseCoinID(), amount)
}

func (a *App) get() *Model {
	if a.model != nil {
		return a.model
	}

	path := []byte{mainPrefix}
	_, enc := a.immutableTree().Get(path)
	if len(enc) == 0 {
		return nil
	}

	model := &Model{}
	if err := rlp.DecodeBytes(enc, model); err != nil {
		panic(fmt.Sprintf("failed to decode legacyApp model at: %s", err))
	}

	a.model = model
	a.model.markDirty = a.markDirty
	return a.model
}

func (a *App) getOrNew() *Model {
	model := a.get()
	if model == nil {
		model = &Model{
			TotalSlashed: big.NewInt(0),
			CoinsCount:   0,
			MaxGas:       0,
			markDirty:    a.markDirty,
		}
		a.model = model
	}

	return model
}

func (a *App) markDirty() {
	a.isDirty = true
}

func (a *App) SetTotalSlashed(amount *big.Int) {
	a.getOrNew().setTotalSlashed(amount)
}

func (a *App) GetCoinsCount() uint32 {
	return a.getOrNew().getCoinsCount()
}

func (a *App) GetNextCoinID() types.CoinID {
	return types.CoinID(a.GetCoinsCount() + 1)
}

func (a *App) SetCoinsCount(count uint32) {
	a.getOrNew().setCoinsCount(count)
}

func (a *App) Export(state *types.AppState) {
	state.MaxGas = a.GetMaxGas()
	state.TotalSlashed = a.GetTotalSlashed().String()
}
