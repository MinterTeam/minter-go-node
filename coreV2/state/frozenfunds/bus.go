package frozenfunds

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Bus struct {
	frozenfunds *FrozenFunds
}

func (b *Bus) AddFrozenFund(height uint64, address types.Address, pubkey *types.Pubkey, candidateID uint32, coin types.CoinID, value *big.Int) {
	b.frozenfunds.AddFund(height, address, pubkey, candidateID, coin, big.NewInt(0).Set(value), 0)
}

func NewBus(frozenfunds *FrozenFunds) *Bus {
	return &Bus{frozenfunds: frozenfunds}
}
