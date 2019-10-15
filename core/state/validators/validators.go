package validators

import (
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
)

type Validators struct {
	info    *Info
	stakes  *Stakes
	rewards *Rewards

	candidatesState *candidates.Candidates

	iavl tree.Tree
}

func NewValidators(iavl tree.Tree, candidatesState *candidates.Candidates) (*Validators, error) {
	return &Validators{iavl: iavl, candidatesState: candidatesState}, nil
}

func (v *Validators) Commit() error {
	panic("implement me")
}

func (v *Validators) SetValidatorPresent(address [20]byte) {
	panic("implement me")
}

func (v *Validators) SetValidatorAbsent(address [20]byte) {
	panic("implement me")
}

func (v *Validators) GetValidators() []ValidatorInfo {
	panic("implement me")
}

func (v *Validators) SetNewValidators(candidates []candidates.Candidate) {
	panic("implement me")
}

func (v *Validators) PunishByzantineValidator(tmAddress [20]byte) {
	panic("implement me")
}

func (v *Validators) Create(ownerAddress types.Address, pubkey []byte, commission int, coin types.CoinSymbol, stake *big.Int) {
	panic("implement me")
}

func (v *Validators) PayRewards() {

}
