package bus

import "github.com/MinterTeam/minter-go-node/coreV2/types"

type Validators interface {
	IsValidator(pubkey types.Pubkey) bool
}
