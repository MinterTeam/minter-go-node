package validators

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"math/big"
)

type Validators struct {
	info    Info
	stakes  Stakes
	rewards Rewards

	db db.DB
}

func NewValidators(db db.DB) (*Validators, error) {
	return &Validators{db: db}, nil
}

func (v *Validators) Commit() error {
	panic("implement me")
}

func (v *Validators) SetValidatorPresent(address [20]byte) {

}

func (v *Validators) SetValidatorAbsent(address [20]byte) {

}

type Info struct {
	data []ValidatorInfo

	isDirty bool
}

type ValidatorInfo struct {
	RewardAddress types.Address
	PubKey        types.Pubkey
	Commission    uint
	AbsentTimes   *types.BitArray

	tmAddress *[20]byte
	toDrop    bool
}

type Stakes struct {
	data []ValidatorStake

	isDirty bool
}

type ValidatorStake *big.Int

type Rewards struct {
	data []ValidatorReward

	isDirty bool
}

type ValidatorReward *big.Int
