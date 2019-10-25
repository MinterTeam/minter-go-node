package validators

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/dao"
	"github.com/MinterTeam/minter-go-node/core/developers"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	compact_db "github.com/klim0v/compact-db"
	"math/big"
)

const (
	mainPrefix        = byte('v')
	totalStakePrefix  = byte('s')
	accumRewardPrefix = byte('r')
)

const ValidatorMaxAbsentWindow = 24

type Validators struct {
	list   []*Validator
	loaded bool

	iavl tree.Tree
	bus  *bus.Bus
}

func NewValidators(bus *bus.Bus, iavl tree.Tree) (*Validators, error) {
	validators := &Validators{iavl: iavl, bus: bus}
	validators.loadValidators()

	return validators, nil
}

func (v *Validators) Commit() error {
	if v.hasDirtyValidators() {
		data, err := rlp.EncodeToBytes(v.list)
		if err != nil {
			return fmt.Errorf("can't encode validators: %v", err)
		}

		path := []byte{mainPrefix}
		v.iavl.Set(path, data)
	}

	for _, val := range v.list {
		if val.isDirty || val.isTotalStakeDirty {
			val.isTotalStakeDirty = false
			path := []byte{mainPrefix}
			path = append(path, val.PubKey.Bytes()...)
			path = append(path, totalStakePrefix)
			v.iavl.Set(path, val.GetTotalBipStake().Bytes())
		}

		if val.isDirty || val.isAccumRewardDirty {
			val.isAccumRewardDirty = false
			path := []byte{mainPrefix}
			path = append(path, val.PubKey.Bytes()...)
			path = append(path, accumRewardPrefix)
			v.iavl.Set(path, val.GetAccumReward().Bytes())
		}
	}

	v.uncheckDirtyValidators()

	return nil
}

func (v *Validators) SetValidatorPresent(height uint64, address types.TmAddress) {
	v.getByTmAddress(address).SetPresent(height)
}

func (v *Validators) SetValidatorAbsent(height uint64, address types.TmAddress) {
	v.getByTmAddress(address).SetAbsent(height)
	// todo: check overflow
}

func (v *Validators) GetValidators() []*Validator {
	return v.list
}

func (v *Validators) SetNewValidators(candidates []candidates.Candidate) {
	panic("implement me")
}

func (v *Validators) PunishByzantineValidator(tmAddress [20]byte) {
	panic("implement me")
}

func (v *Validators) Create(ownerAddress types.Address, pubkey types.Pubkey, commission uint, coin types.CoinSymbol, stake *big.Int) {
	val := &Validator{
		OwnerAddress:  ownerAddress,
		RewardAddress: ownerAddress,
		PubKey:        pubkey,
		Commission:    commission,
		AbsentTimes:   types.NewBitArray(ValidatorMaxAbsentWindow),
		totalStake:    stake,
		accumReward:   big.NewInt(0),
	}
	val.setTmAddress()
	v.list = append(v.list, val)
}

func (v *Validators) PayRewards(height uint64) {
	vals := v.GetValidators()
	for _, validator := range vals {
		if validator.GetAccumReward().Cmp(types.Big0) == 1 {
			totalReward := big.NewInt(0).Set(validator.GetAccumReward())
			remainder := big.NewInt(0).Set(validator.GetAccumReward())

			// pay commission to DAO
			DAOReward := big.NewInt(0).Set(totalReward)
			DAOReward.Mul(DAOReward, big.NewInt(int64(dao.Commission)))
			DAOReward.Div(DAOReward, big.NewInt(100))
			v.bus.Accounts().AddBalance(dao.Address, types.GetBaseCoin(), DAOReward)
			remainder.Sub(remainder, DAOReward)
			v.bus.Events().AddEvent(uint32(height), compact_db.RewardEvent{
				Role:            compact_db.RoleDAO,
				Address:         dao.Address,
				Amount:          DAOReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			// pay commission to Developers
			DevelopersReward := big.NewInt(0).Set(totalReward)
			DevelopersReward.Mul(DevelopersReward, big.NewInt(int64(developers.Commission)))
			DevelopersReward.Div(DevelopersReward, big.NewInt(100))
			v.bus.Accounts().AddBalance(developers.Address, types.GetBaseCoin(), DevelopersReward)
			remainder.Sub(remainder, DevelopersReward)
			v.bus.Events().AddEvent(uint32(height), compact_db.RewardEvent{
				Role:            compact_db.RoleDevelopers,
				Address:         developers.Address,
				Amount:          DevelopersReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			totalReward.Sub(totalReward, DevelopersReward)
			totalReward.Sub(totalReward, DAOReward)

			// pay commission to validator
			validatorReward := big.NewInt(0).Set(totalReward)
			validatorReward.Mul(validatorReward, big.NewInt(int64(validator.Commission)))
			validatorReward.Div(validatorReward, big.NewInt(100))
			totalReward.Sub(totalReward, validatorReward)
			v.bus.Accounts().AddBalance(validator.RewardAddress, types.GetBaseCoin(), validatorReward)
			remainder.Sub(remainder, validatorReward)
			v.bus.Events().AddEvent(uint32(height), compact_db.RewardEvent{
				Role:            compact_db.RoleValidator,
				Address:         validator.RewardAddress,
				Amount:          validatorReward.Bytes(),
				ValidatorPubKey: validator.PubKey,
			})

			stakes := v.bus.Candidates().GetStakes(validator.PubKey)

			// pay rewards
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

				v.bus.Accounts().AddBalance(stake.Owner, types.GetBaseCoin(), reward)
				remainder.Sub(remainder, reward)

				v.bus.Events().AddEvent(uint32(height), compact_db.RewardEvent{
					Role:            compact_db.RoleDelegator,
					Address:         stake.Owner,
					Amount:          reward.Bytes(),
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

func (v *Validators) getByTmAddress(address types.TmAddress) *Validator {
	for _, val := range v.list {
		if val.tmAddress == address {
			return val
		}
	}

	return nil
}

func (v *Validators) loadValidators() {
	if v.loaded {
		return
	}

	v.loaded = true

	path := []byte{mainPrefix}
	_, enc := v.iavl.Get(path)
	if len(enc) == 0 {
		v.list = nil
		return
	}

	var validators []*Validator
	if err := rlp.DecodeBytes(enc, validators); err != nil {
		panic(fmt.Sprintf("failed to decode validators: %s", err))
		return
	}

	v.list = validators
	for _, validator := range validators {
		// load total stake
		path = append([]byte{mainPrefix}, validator.PubKey.Bytes()...)
		path = append(path, totalStakePrefix)
		_, enc = v.iavl.Get(path)
		if len(enc) == 0 {
			validator.totalStake = big.NewInt(0)
		} else {
			validator.totalStake = big.NewInt(0).SetBytes(enc)
		}

		// load accum reward
		path = append([]byte{mainPrefix}, validator.PubKey.Bytes()...)
		path = append(path, accumRewardPrefix)
		_, enc = v.iavl.Get(path)
		if len(enc) == 0 {
			validator.accumReward = big.NewInt(0)
		} else {
			validator.accumReward = big.NewInt(0).SetBytes(enc)
		}

		validator.setTmAddress()
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
