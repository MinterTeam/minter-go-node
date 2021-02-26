package halts

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
)

type Bus struct {
	halts *HaltBlocks
}

func (b *Bus) AddHaltBlock(height uint64, pubkey types.Pubkey) {
	b.halts.AddHaltBlock(height, pubkey)
}

func NewBus(halts *HaltBlocks) *Bus {
	return &Bus{halts: halts}
}
