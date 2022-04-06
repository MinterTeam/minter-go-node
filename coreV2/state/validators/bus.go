package validators

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
)

type Bus struct {
	validators *Validators
}

func NewBus(validators *Validators) *Bus {
	return &Bus{validators: validators}
}

func (b *Bus) IsValidator(pubkey types.Pubkey) bool {
	return b.validators.IsValidator(pubkey)
}
func (b *Bus) Count() int {
	return b.validators.Count()
}
