package app

import (
	db "github.com/tendermint/tm-db"
	"math/big"
)

type App struct {
	isDirty bool

	db db.DB
}

func NewApp(db db.DB) (*App, error) {
	return &App{db: db}, nil
}

func (v *App) Commit() error {
	panic("implement me")
}

func (v *App) SetMaxGas(gas uint64) {
	panic("implement me")
}

func (v *App) AddTotalSlashed(amount *big.Int) {
	panic("implement me")
}

func (v *App) GetMaxGas() uint64 {
	panic("implement me")
}
