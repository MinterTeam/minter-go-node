package bus

import "github.com/MinterTeam/minter-go-node/coreV2/types"

type HaltBlocks interface {
	AddHaltBlock(uint64, types.Pubkey)
}
