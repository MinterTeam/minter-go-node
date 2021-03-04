package validators

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"math/big"
	"sync"
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

	lock sync.RWMutex
	bus  *bus.Bus
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
	v.lock.RLock()
	defer v.lock.RUnlock()

	return v.toDrop
}

func (v *Validator) SetAccumReward(value *big.Int) {
	v.lock.Lock()

	if v.accumReward.Cmp(value) == 0 {
		v.lock.Unlock()
		return
	}
	v.isAccumRewardDirty = true
	oldAcc := big.NewInt(0).Set(v.accumReward)
	v.accumReward = big.NewInt(0).Set(value)
	v.lock.Unlock()

	v.bus.Checker().AddCoin(types.GetBaseCoinID(), big.NewInt(0).Sub(value, oldAcc), "reward")
}

func (v *Validator) GetAccumReward() *big.Int {
	v.lock.RLock()
	defer v.lock.RUnlock()

	return big.NewInt(0).Set(v.accumReward)
}

// GetAddress returns tendermint-address of a validator
func (v *Validator) GetAddress() types.TmAddress {
	v.lock.RLock()
	defer v.lock.RUnlock()

	return v.tmAddress
}

// GetTotalBipStake returns total bip stake
func (v *Validator) GetTotalBipStake() *big.Int {
	v.lock.RLock()
	defer v.lock.RUnlock()

	return big.NewInt(0).Set(v.totalStake)
}

// SetTotalBipStake sets total bip stake
func (v *Validator) SetTotalBipStake(value *big.Int) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if v.totalStake.Cmp(value) == 0 {
		return
	}
	v.isTotalStakeDirty = true
	v.totalStake = big.NewInt(0).Set(value)
}

func (v *Validator) AddAccumReward(amount *big.Int) {
	v.lock.Lock()
	reward := big.NewInt(0).Set(v.accumReward)
	v.lock.Unlock()

	v.SetAccumReward(big.NewInt(0).Add(reward, amount))
}

func (v *Validator) CountAbsentTimes() int {
	count := 0

	for i := 0; i < validatorMaxAbsentWindow; i++ {
		v.lock.RLock()
		if v.AbsentTimes.GetIndex(i) {
			count++
		}
		v.lock.RUnlock()
	}

	return count
}

func (v *Validator) setTmAddress() {
	// set tm address
	v.lock.RLock()
	add := ed25519.PubKey(v.PubKey[:]).Address()
	v.lock.RUnlock()

	var address types.TmAddress
	copy(address[:], add.Bytes())

	v.lock.Lock()
	v.tmAddress = address
	v.lock.Unlock()
}

func (v *Validator) SetPresent(height uint64) {
	index := int(height) % validatorMaxAbsentWindow

	v.lock.Lock()
	defer v.lock.Unlock()

	if v.AbsentTimes.GetIndex(index) {
		v.isDirty = true
	}
	v.AbsentTimes.SetIndex(index, false)
}

func (v *Validator) SetAbsent(height uint64) {
	index := int(height) % validatorMaxAbsentWindow

	v.lock.Lock()
	defer v.lock.Unlock()

	if !v.AbsentTimes.GetIndex(index) {
		v.isDirty = true
	}
	v.AbsentTimes.SetIndex(index, true)
}
