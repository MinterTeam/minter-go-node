package frozenfunds

import "github.com/MinterTeam/minter-go-node/core/types"

type Bus struct {
	frozenfunds *FrozenFunds
}

func (b *Bus) DeleteCoin(height uint64, coin types.CoinSymbol) {
	b.frozenfunds.DeleteCoin(height, coin)
}

func NewBus(frozenfunds *FrozenFunds) *Bus {
	return &Bus{frozenfunds: frozenfunds}
}
