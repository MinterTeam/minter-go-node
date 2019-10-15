package validators

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

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

func (info *ValidatorInfo) CountAbsentTimes() int {
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
