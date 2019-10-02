package validators

import (
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
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

func (info *ValidatorInfo) IsToDrop() bool {
	return info.toDrop
}

func (info *ValidatorInfo) SetAccumReward(value *big.Int) {
	panic("implement me")
}

func (info *ValidatorInfo) GetAccumReward() *big.Int {
	panic("implement me")
}

func (info *ValidatorInfo) GetAddress() [20]byte {
	panic("implement me")
}

func (info *ValidatorInfo) GetTotalBipStake() *big.Int {
	panic("implement me")
}

func (info *ValidatorInfo) AddAccumReward(amount *big.Int) {
	panic("implement me")
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
