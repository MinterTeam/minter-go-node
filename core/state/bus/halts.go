package bus

import "github.com/MinterTeam/minter-go-node/core/types"

type HaltBlocks interface {
	AddHaltBlock(uint64, types.Pubkey)
}
