package validators

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
)

type Validator struct {
	PubKey      types.Pubkey
	AbsentTimes *types.BitArray

	totalStake  *big.Int
	accumReward *big.Int

	isDirty            bool
	isTotalStakeDirty  bool
	isAccumRewardDirty bool

	tmAddress types.TmAddress
	toDrop    bool

	bus *bus.Bus
}

func NewValidator(pubKey types.Pubkey, absentTimes *types.BitArray, totalStake *big.Int, accumReward *big.Int, isDirty bool, isTotalStakeDirty bool, isAccumRewardDirty bool, bus *bus.Bus) *Validator {
	val := &Validator{
		PubKey:             pubKey,
		AbsentTimes:        absentTimes,
		totalStake:         totalStake,
		accumReward:        accumReward,
		isDirty:            isDirty,
		isTotalStakeDirty:  isTotalStakeDirty,
		isAccumRewardDirty: isAccumRewardDirty,
		bus:                bus,
	}
	val.setTmAddress()
	return val
}

func (v *Validator) IsToDrop() bool {
	return v.toDrop
}

func (v *Validator) SetAccumReward(value *big.Int) {
	v.bus.Checker().AddCoin(types.GetBaseCoin(), big.NewInt(0).Sub(value, v.accumReward), "reward")
	v.accumReward = big.NewInt(0).Set(value)
	v.isAccumRewardDirty = true
}

func (v *Validator) GetAccumReward() *big.Int {
	return big.NewInt(0).Set(v.accumReward)
}

func (v *Validator) GetAddress() types.TmAddress {
	return v.tmAddress
}

func (v *Validator) GetTotalBipStake() *big.Int {
	return big.NewInt(0).Set(v.totalStake)
}

func (v *Validator) SetTotalBipStake(value *big.Int) {
	v.totalStake = value
	v.isTotalStakeDirty = true
}

func (v *Validator) AddAccumReward(amount *big.Int) {
	v.SetAccumReward(big.NewInt(0).Add(v.accumReward, amount))
}

func (v *Validator) CountAbsentTimes() int {
	count := 0

	for i := 0; i < ValidatorMaxAbsentWindow; i++ {
		if v.AbsentTimes.GetIndex(i) {
			count++
		}
	}

	return count
}

func (v *Validator) setTmAddress() {
	// set tm address
	var pubkey ed25519.PubKeyEd25519
	copy(pubkey[:], v.PubKey[:])

	var address types.TmAddress
	copy(address[:], pubkey.Address().Bytes())

	v.tmAddress = address
}

func (v *Validator) SetPresent(height uint64) {
	v.AbsentTimes.SetIndex(int(height)%ValidatorMaxAbsentWindow, false)
	v.isDirty = true
}

func (v *Validator) SetAbsent(height uint64) {
	v.AbsentTimes.SetIndex(int(height)%ValidatorMaxAbsentWindow, true)
	v.isDirty = true
}
