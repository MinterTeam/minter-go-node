package app

import "math/big"

type Bus struct {
	app *App
}

func (b *Bus) AddTotalSlashed(amount *big.Int) {
	b.app.AddTotalSlashed(amount)
}

func NewBus(app *App) *Bus {
	return &Bus{app: app}
}
