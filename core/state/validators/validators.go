package validators

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/dao"
	"github.com/MinterTeam/minter-go-node/core/developers"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/upgrades"
	"github.com/tendermint/iavl"
	"sync/atomic"

	"math/big"
)

const (
	mainPrefix        = byte('v')
	totalStakePrefix  = byte('s')
	accumRewardPrefix = byte('r')
)

const (
	validatorMaxAbsentWindow = 24
	validatorMaxAbsentTimes  = 12
)

// Validators struct is a store of Validators state
type Validators struct {
	list    []*Validator
	removed map[types.Pubkey]struct{}
	loaded  bool

	db  atomic.Value
	bus *bus.Bus
}

// RValidators interface represents Validator state
type RValidators interface {
	GetValidators() []*Validator
	Export(state *types.AppState)
	GetByPublicKey(pubKey types.Pubkey) *Validator
	LoadValidators()
	GetByTmAddress(address types.TmAddress) *Validator
}

// NewValidators returns newly created Validators state with a given bus and iavl
func NewValidators(bus *bus.Bus, db *iavl.ImmutableTree) *Validators {
	immutableTree := atomic.Value{}
	loaded := false
	if db != nil {
		immutableTree.Store(db)
	} else {
		loaded = true
	}
	validators := &Validators{db: immutableTree, bus: bus, loaded: loaded}

	return validators
}

func (v *Validators) immutableTree() *iavl.ImmutableTree {
	db := v.db.Load()
	if db == nil {
		return nil
	}
	return db.(*iavl.ImmutableTree)
}

func (v *Validators) SetImmutableTree(immutableTree *iavl.ImmutableTree) {
	if v.immutableTree() == nil && v.loaded {
		v.loaded = false
	}
	v.db.Store(immutableTree)
}

// Commit writes changes to iavl, may return an error
func (v *Validators) Commit(db *iavl.MutableTree) error {
	if v.hasDirtyValidators() {
		data, err := rlp.EncodeToBytes(v.list)
		if err != nil {
			return fmt.Errorf("can't encode validators: %v", err)
		}

		path := []byte{mainPrefix}
		db.Set(path, data)
	}

	for _, val := range v.list {
		if val.isDirty || val.isTotalStakeDirty {
			val.isTotalStakeDirty = false
			path := []byte{mainPrefix}
			path = append(path, val.PubKey.Bytes()...)
			path = append(path, totalStakePrefix)
			db.Set(path, val.GetTotalBipStake().Bytes())
		}

		if val.isDirty || val.isAccumRewardDirty {
			val.isAccumRewardDirty = false
			path := []byte{mainPrefix}
			path = append(path, val.PubKey.Bytes()...)
			path = append(path, accumRewardPrefix)
			db.Set(path, val.GetAccumReward().Bytes())
		}
	}

	for pubkey := range v.removed {
		path := append([]byte{mainPrefix}, pubkey.Bytes()...)
		v.iavl.Remove(append(path, totalStakePrefix))
		v.iavl.Remove(append(path, accumRewardPrefix))
	}
	v.removed = map[types.Pubkey]struct{}{}

	v.uncheckDirtyValidators()

	return nil
}

// SetValidatorPresent marks validator as present at current height
func (v *Validators) SetValidatorPresent(height uint64, address types.TmAddress) {
	validator := v.GetByTmAddress(address)
	if validator == nil {
		return
	}
	validator.SetPresent(height)
}

// SetValidatorAbsent marks validator as absent at current height
// if validator misses signs of more than validatorMaxAbsentTimes, it will receive penalty and will be swithed off
func (v *Validators) SetValidatorAbsent(height uint64, address types.TmAddress) {
	validator := v.GetByTmAddress(address)
	if validator == nil {
		return
	}
	validator.SetAbsent(height)

	if validator.CountAbsentTimes() > validatorMaxAbsentTimes {
		if !upgrades.IsGraceBlock(height) {
			v.punishValidator(height, address)
		}

		v.turnValidatorOff(address)
	}
}

// GetValidators returns list of validators
func (v *Validators) GetValidators() []*Validator {
	return v.list
}

// SetNewValidators updated validators list with new candidates
func (v *Validators) SetNewValidators(candidates []candidates.Candidate) {
	old := v.GetValidators()

	oldValidatorsForRemove := map[types.Pubkey]struct{}{}
	for _, oldVal := range old {
		oldValidatorsForRemove[oldVal.PubKey] = struct{}{}
	}

	var newVals []*Validator
	for _, candidate := range candidates {
		accumReward := big.NewInt(0)
		absentTimes := types.NewBitArray(validatorMaxAbsentWindow)

		for _, oldVal := range old {
			if oldVal.GetAddress() == candidate.GetTmAddress() {
				accumReward = oldVal.accumReward
				absentTimes = oldVal.AbsentTimes
				delete(oldValidatorsForRemove, oldVal.PubKey)
			}
		}

		newVals = append(newVals, &Validator{
			PubKey:             candidate.PubKey,
			AbsentTimes:        absentTimes,
			totalStake:         candidate.GetTotalBipStake(),
			accumReward:        accumReward,
			isDirty:            true,
			isTotalStakeDirty:  true,
			isAccumRewardDirty: true,
			tmAddress:          candidate.GetTmAddress(),
			bus:                v.bus,
		})
	}

	v.removed = oldValidatorsForRemove
	v.SetValidators(newVals)
}

// PunishByzantineValidator find validator with given tmAddress and punishes it:
// 1. Set total stake 0
// 2. Drop validator
func (v *Validators) PunishByzantineValidator(tmAddress [20]byte) {
	validator := v.GetByTmAddress(tmAddress)
	if validator != nil {
		validator.SetTotalBipStake(big.NewInt(0))
		validator.toDrop = true
		validator.isDirty = true
	}
}

// Create creates a new validator with given params and adds it to state
func (v *Validators) Create(pubkey types.Pubkey, stake *big.Int) {
	val := &Validator{
		PubKey:             pubkey,
		AbsentTimes:        types.NewBitArray(validatorMaxAbsentWindow),
		totalStake:         big.NewInt(0).Set(stake),
		accumReward:        big.NewInt(0),
		isDirty:            true,
		isTotalStakeDirty:  true,
		isAccumRewardDirty: true,
	}

	val.setTmAddress()
	v.list = append(v.list, val)
}

// PayRewards distributes accumulated rewards between validator, delegators, DAO and developers addresses
func (v *Validators) PayRewards(height uint64) {
	vals := v.GetValidators()
	for _, validator := range vals {
		if validator.GetAccumReward().Cmp(types.Big0) == 1 {
			candidate := v.bus.Candidates().GetCandidate(validator.PubKey)

			totalReward := big.NewInt(0).Set(validator.GetAccumReward())
			remainder := big.NewInt(0).Set(validator.GetAccumReward())

			// pay commission to DAO
			DAOReward := big.NewInt(0).Set(totalReward)
			DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
			DAOReward.Div(DAOReward, big.NewInt(100))
			v.bus.Accounts().AddBalance(dao.Address, types.GetBaseCoinID(), DAOReward)
			remainder.Sub(remainder, DAOReward)
			v.bus.Events().AddEvent(uint32(height), &eventsdb.RewardEvent{
				Role:            eventsdb.RoleDAO.String(),
				Address:         dao.Address,
				Amount:          DAOReward.String(),
				ValidatorPubKey: validator.PubKey,
			})

			// pay commission to Developers
			DevelopersReward := big.NewInt(0).Set(totalReward)
			DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
			DevelopersReward.Div(DevelopersReward, big.NewInt(100))
			v.bus.Accounts().AddBalance(developers.Address, types.GetBaseCoinID(), DevelopersReward)
			remainder.Sub(remainder, DevelopersReward)
			v.bus.Events().AddEvent(uint32(height), &eventsdb.RewardEvent{
				Role:            eventsdb.RoleDevelopers.String(),
				Address:         developers.Address,
				Amount:          DevelopersReward.String(),
				ValidatorPubKey: validator.PubKey,
			})

			totalReward.Sub(totalReward, DevelopersReward)
			totalReward.Sub(totalReward, DAOReward)

			// pay commission to validator
			validatorReward := big.NewInt(0).Set(totalReward)
			validatorReward.Mul(validatorReward, big.NewInt(int64(candidate.Commission)))
			validatorReward.Div(validatorReward, big.NewInt(100))
			totalReward.Sub(totalReward, validatorReward)
			v.bus.Accounts().AddBalance(candidate.RewardAddress, types.GetBaseCoinID(), validatorReward)
			remainder.Sub(remainder, validatorReward)
			v.bus.Events().AddEvent(uint32(height), &eventsdb.RewardEvent{
				Role:            eventsdb.RoleValidator.String(),
				Address:         candidate.RewardAddress,
				Amount:          validatorReward.String(),
				ValidatorPubKey: validator.PubKey,
			})

			stakes := v.bus.Candidates().GetStakes(validator.PubKey)
			for _, stake := range stakes {
				if stake.BipValue.Cmp(big.NewInt(0)) == 0 {
					continue
				}

				reward := big.NewInt(0).Set(totalReward)
				reward.Mul(reward, stake.BipValue)

				reward.Div(reward, validator.GetTotalBipStake())
				if reward.Cmp(types.Big0) < 1 {
					continue
				}

				v.bus.Accounts().AddBalance(stake.Owner, types.GetBaseCoinID(), reward)
				remainder.Sub(remainder, reward)

				v.bus.Events().AddEvent(uint32(height), &eventsdb.RewardEvent{
					Role:            eventsdb.RoleDelegator.String(),
					Address:         stake.Owner,
					Amount:          reward.String(),
					ValidatorPubKey: validator.PubKey,
				})
			}

			validator.SetAccumReward(big.NewInt(0))

			if remainder.Cmp(big.NewInt(0)) > -1 {
				v.bus.App().AddTotalSlashed(remainder)
			} else {
				panic(fmt.Sprintf("Negative remainder: %s", remainder.String()))
			}
		}
	}
}

// GetByTmAddress finds and returns validator with given tendermint-address
func (v *Validators) GetByTmAddress(address types.TmAddress) *Validator {
	for _, val := range v.list {
		if val.tmAddress == address {
			return val
		}
	}

	return nil
}

// GetByPublicKey finds and returns validator
func (v *Validators) GetByPublicKey(pubKey types.Pubkey) *Validator {
	for _, val := range v.list {
		if val.PubKey == pubKey {
			return val
		}
	}

	return nil
}

// LoadValidators loads only list of validators (for read)
func (v *Validators) LoadValidators() {
	if v.loaded {
		return
	}

	v.loaded = true

	path := []byte{mainPrefix}
	_, enc := v.immutableTree().Get(path)
	if len(enc) == 0 {
		v.list = nil
		return
	}

	var validators []*Validator
	if err := rlp.DecodeBytes(enc, &validators); err != nil {
		panic(fmt.Sprintf("failed to decode validators: %s", err))
	}

	v.list = validators
	for _, validator := range validators {
		// load total stake
		path = append([]byte{mainPrefix}, validator.PubKey.Bytes()...)
		path = append(path, totalStakePrefix)
		_, enc = v.immutableTree().Get(path)
		if len(enc) == 0 {
			validator.totalStake = big.NewInt(0)
		} else {
			validator.totalStake = big.NewInt(0).SetBytes(enc)
		}

		// load accum reward
		path = append([]byte{mainPrefix}, validator.PubKey.Bytes()...)
		path = append(path, accumRewardPrefix)
		_, enc = v.immutableTree().Get(path)
		if len(enc) == 0 {
			validator.accumReward = big.NewInt(0)
		} else {
			validator.accumReward = big.NewInt(0).SetBytes(enc)
		}

		validator.setTmAddress()
		validator.bus = v.bus
	}
}

func (v *Validators) hasDirtyValidators() bool {
	for _, val := range v.list {
		if val.isDirty {
			return true
		}
	}

	return false
}

func (v *Validators) uncheckDirtyValidators() {
	for _, val := range v.list {
		val.isDirty = false
	}
}

func (v *Validators) punishValidator(height uint64, tmAddress types.TmAddress) {
	validator := v.GetByTmAddress(tmAddress)

	totalStake := v.bus.Candidates().Punish(height, tmAddress)
	validator.SetTotalBipStake(totalStake)
}

// SetValidators updates validators list
func (v *Validators) SetValidators(vals []*Validator) {
	v.list = vals
}

// Export exports all data to the given state
func (v *Validators) Export(state *types.AppState) {
	v.LoadValidators()

	for _, val := range v.GetValidators() {
		state.Validators = append(state.Validators, types.Validator{
			TotalBipStake: val.GetTotalBipStake().String(),
			PubKey:        val.PubKey,
			AccumReward:   val.GetAccumReward().String(),
			AbsentTimes:   val.AbsentTimes,
		})
	}
}

// SetToDrop marks given validator as inactive for dropping it in the next block
func (v *Validators) SetToDrop(pubkey types.Pubkey) {
	vals := v.GetValidators()
	for _, val := range vals {
		if val.PubKey == pubkey {
			val.toDrop = true
		}
	}
}

func (v *Validators) turnValidatorOff(tmAddress types.TmAddress) {
	validator := v.GetByTmAddress(tmAddress)
	validator.AbsentTimes = types.NewBitArray(validatorMaxAbsentWindow)
	validator.toDrop = true
	validator.isDirty = true

	v.bus.Candidates().SetOffline(validator.PubKey)
}
