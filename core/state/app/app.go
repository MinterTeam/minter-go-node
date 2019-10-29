package app

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
)

const mainPrefix = 'd'

type App struct {
	model   *Model
	isDirty bool

	bus  *bus.Bus
	iavl tree.Tree
}

func NewApp(stateBus *bus.Bus, iavl tree.Tree) (*App, error) {
	app := &App{bus: stateBus, iavl: iavl}
	app.bus.SetApp(NewBus(app))

	return app, nil
}

func (v *App) Commit() error {
	if !v.isDirty {
		return nil
	}

	data, err := rlp.EncodeToBytes(v.model)
	if err != nil {
		return fmt.Errorf("can't encode app model: %s", err)
	}

	path := []byte{mainPrefix}
	v.iavl.Set(path, data)

	return nil
}

func (v *App) GetMaxGas() uint64 {
	model := v.getOrNew()

	return model.getMaxGas()
}

func (v *App) SetMaxGas(gas uint64) {
	model := v.getOrNew()
	model.setMaxGas(gas)
}

func (v *App) GetTotalSlashed() *big.Int {
	model := v.getOrNew()

	return model.getTotalSlashed()
}

func (v *App) AddTotalSlashed(amount *big.Int) {
	model := v.getOrNew()
	model.setTotalSlashed(big.NewInt(0).Add(model.getTotalSlashed(), amount))
}

func (v *App) get() *Model {
	if v.model != nil {
		return v.model
	}

	path := []byte{mainPrefix}
	_, enc := v.iavl.Get(path)
	if len(enc) == 0 {
		return nil
	}

	model := &Model{}
	if err := rlp.DecodeBytes(enc, model); err != nil {
		panic(fmt.Sprintf("failed to decode app model at: %s", err))
		return nil
	}

	v.model = model
	v.model.markDirty = v.markDirty
	return v.model
}

func (v *App) getOrNew() *Model {
	model := v.get()
	if model == nil {
		model = &Model{
			TotalSlashed: big.NewInt(0),
			MaxGas:       0,
			markDirty:    v.markDirty,
		}
		v.model = model
	}

	return model
}

func (v *App) markDirty() {
	v.isDirty = true
}

func (v *App) SetTotalSlashed(amount *big.Int) {
	v.getOrNew().setTotalSlashed(amount)
}

func (v *App) Export(state *types.AppState, height uint64) {
	state.MaxGas = v.GetMaxGas()
	state.TotalSlashed = v.GetTotalSlashed()
	state.StartHeight = height
}
