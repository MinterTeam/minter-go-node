package validators

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/dao"
	"github.com/MinterTeam/minter-go-node/coreV2/developers"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"github.com/MinterTeam/minter-go-node/coreV2/state/app"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestValidators_GetValidators(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	validators.Create([32]byte{1}, big.NewInt(1000000))
	validators.Create([32]byte{2}, big.NewInt(2000000))
	getValidators := validators.GetValidators()
	if len(getValidators) != 2 {
		t.Fatal("count of validators not equal 2")
	}
	if getValidators[0].PubKey != [32]byte{1} {
		t.Fatal("validator public_key invalid")
	}
	if getValidators[0].totalStake.String() != "1000000" {
		t.Fatal("validator total_stake invalid")
	}
	if getValidators[1].PubKey != [32]byte{2} {
		t.Fatal("validator public_key invalid")
	}
	if getValidators[1].totalStake.String() != "2000000" {
		t.Fatal("validator total_stake invalid")
	}
}

func TestValidators_GetByPublicKey(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	validators.Create([32]byte{1}, big.NewInt(1000000))
	validator := validators.GetByPublicKey([32]byte{1})
	if validator == nil {
		t.Fatal("validator not found")
	}
	if validator.PubKey != [32]byte{1} {
		t.Fatal("validator public_key invalid")
	}
	if validator.totalStake.String() != "1000000" {
		t.Fatal("validator total_stake invalid")
	}
}

func TestValidators_GetByTmAddress(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	validators.Create([32]byte{1}, big.NewInt(1000000))
	validator := validators.GetByPublicKey([32]byte{1})
	if validator == nil {
		t.Fatal("validator not found")
	}
	vldtr := validators.GetByTmAddress(validator.tmAddress)

	if vldtr.PubKey != [32]byte{1} {
		t.Fatal("validator public_key invalid")
	}
	if vldtr.totalStake.String() != "1000000" {
		t.Fatal("validator total_stake invalid")
	}
}

func TestValidators_PunishByzantineValidator(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	validators.Create([32]byte{1}, big.NewInt(1000000))
	validator := validators.GetByPublicKey([32]byte{1})
	if validator == nil {
		t.Fatal("validator not found")
	}

	validators.PunishByzantineValidator(validator.tmAddress)

	if validator.totalStake.String() != "0" {
		t.Fatal("validator total_stake invalid")
	}
}

func TestValidators_LoadValidators(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	validators := NewValidators(b, mutableTree.GetLastImmutable())

	newValidator := NewValidator(
		[32]byte{1},
		types.NewBitArray(ValidatorMaxAbsentWindow),
		big.NewInt(1000000),
		big.NewInt(0),
		true,
		true,
		true,
		b)
	newValidator.AddAccumReward(big.NewInt(10))
	validators.SetValidators([]*Validator{newValidator})

	validators.Create([32]byte{2}, big.NewInt(2000000))

	_, _, err := mutableTree.Commit(validators)
	if err != nil {
		t.Fatal(err)
	}

	validators = NewValidators(b, mutableTree.GetLastImmutable())

	validators.LoadValidators()

	getValidators := validators.GetValidators()
	if len(getValidators) != 2 {
		t.Fatal("count of validators not equal 2")
	}
	if getValidators[0].PubKey != [32]byte{1} {
		t.Fatal("validator public_key invalid")
	}
	if getValidators[0].totalStake.String() != "1000000" {
		t.Fatal("validator total_stake invalid")
	}
	if getValidators[1].PubKey != [32]byte{2} {
		t.Fatal("validator public_key invalid")
	}
	if getValidators[1].totalStake.String() != "2000000" {
		t.Fatal("validator total_stake invalid")
	}
}

func TestValidators_SetValidators(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	newValidator := NewValidator(
		[32]byte{1},
		types.NewBitArray(ValidatorMaxAbsentWindow),
		big.NewInt(1000000),
		big.NewInt(0),
		true,
		true,
		true,
		b)
	validators.SetValidators([]*Validator{newValidator})

	validator := validators.GetByPublicKey([32]byte{1})
	if validator == nil {
		t.Fatal("validator not found")
	}
	if validator.PubKey != [32]byte{1} {
		t.Fatal("validator public_key invalid")
	}
	if validator.totalStake.String() != "1000000" {
		t.Fatal("validator total_stake invalid")
	}
}

func TestValidators_PayRewards(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	b.SetChecker(checker.NewChecker(b))
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	appBus := app.NewApp(b, mutableTree.GetLastImmutable())
	b.SetApp(appBus)
	validators := NewValidators(b, mutableTree.GetLastImmutable())
	newValidator := NewValidator(
		[32]byte{4},
		types.NewBitArray(ValidatorMaxAbsentWindow),
		big.NewInt(1000000),
		big.NewInt(10),
		true,
		true,
		true,
		b)
	validators.SetValidators([]*Validator{newValidator})
	validator := validators.GetByPublicKey([32]byte{4})
	if validator == nil {
		t.Fatal("validator not found")
	}
	validator.AddAccumReward(big.NewInt(90))
	candidatesS := candidates.NewCandidates(b, mutableTree.GetLastImmutable())

	candidatesS.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidatesS.SetOnline([32]byte{4})
	candidatesS.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "1000000000000000000000",
			BipValue: "1000000000000000000000",
		},
	}, nil)
	candidatesS.RecalculateStakes(0)
	validators.SetNewValidators(candidatesS.GetNewCandidates(1))

	validators.PayRewards()

	if accs.GetBalance([20]byte{1}, 0).String() != "72" {
		t.Fatal("delegate did not receive the award")
	}
	if accs.GetBalance([20]byte{2}, 0).String() != "8" {
		t.Fatal("rewards_address did not receive the award")
	}

	if accs.GetBalance(dao.Address, 0).String() != "10" {
		t.Fatal("dao_address did not receive the award")
	}
	if accs.GetBalance(developers.Address, 0).String() != "10" {
		t.Fatal("developers_address did not receive the award")
	}
}

func TestValidators_SetValidatorAbsent(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	b.SetChecker(checker.NewChecker(b))
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	appBus := app.NewApp(b, mutableTree.GetLastImmutable())
	b.SetApp(appBus)
	validators := NewValidators(b, mutableTree.GetLastImmutable())
	newValidator := NewValidator(
		[32]byte{4},
		types.NewBitArray(ValidatorMaxAbsentWindow),
		big.NewInt(1000000),
		big.NewInt(100),
		true,
		true,
		true,
		b)
	validators.SetValidators([]*Validator{newValidator})

	candidatesS := candidates.NewCandidates(b, mutableTree.GetLastImmutable())

	candidatesS.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidatesS.SetOnline([32]byte{4})
	candidatesS.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "1000000000000000000000",
			BipValue: "1000000000000000000000",
		},
	}, nil)
	candidatesS.RecalculateStakes(0)
	validators.SetNewValidators(candidatesS.GetNewCandidates(1))

	validator := validators.GetByPublicKey([32]byte{4})
	if validator == nil {
		t.Fatal("validator not found")
	}
	for i := uint64(0); i < validatorMaxAbsentTimes+1; i++ {
		validators.SetValidatorAbsent(i, validator.tmAddress)
	}
	if !validator.IsToDrop() {
		t.Fatal("validator not drop")
	}
}
func TestValidators_SetValidatorPresent(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	validators.Create([32]byte{4}, big.NewInt(1000000))

	validator := validators.GetByPublicKey([32]byte{4})
	if validator == nil {
		t.Fatal("validator not found")
	}

	validators.SetValidatorAbsent(0, validator.tmAddress)

	if validator.AbsentTimes.String() != "BA{24:x_______________________}" {
		t.Fatal("validator has not absent")
	}

	validators.SetValidatorPresent(0, validator.tmAddress)

	if validator.AbsentTimes.String() != "BA{24:________________________}" {
		t.Fatal("validator has absent")
	}
}

func TestValidators_SetToDrop(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()

	validators := NewValidators(b, mutableTree.GetLastImmutable())

	validators.Create([32]byte{4}, big.NewInt(1000000))

	validator := validators.GetByPublicKey([32]byte{4})
	if validator == nil {
		t.Fatal("validator not found")
	}

	if validator.toDrop {
		t.Fatal("default validator set to drop")
	}
	validators.SetToDrop([32]byte{4})
	if !validator.toDrop {
		t.Fatal("validator not set to drop")
	}
}

func TestValidators_Export(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	accs := accounts.NewAccounts(b, mutableTree.GetLastImmutable())

	b.SetAccounts(accounts.NewBus(accs))
	b.SetChecker(checker.NewChecker(b))
	b.SetEvents(eventsdb.NewEventsStore(db.NewMemDB()))
	appBus := app.NewApp(b, mutableTree.GetLastImmutable())

	b.SetApp(appBus)
	validators := NewValidators(b, mutableTree.GetLastImmutable())
	newValidator := NewValidator(
		[32]byte{4},
		types.NewBitArray(ValidatorMaxAbsentWindow),
		helpers.BipToPip(big.NewInt(1000000)),
		big.NewInt(100),
		true,
		true,
		true,
		b)
	validators.SetValidators([]*Validator{newValidator})

	candidatesS := candidates.NewCandidates(b, mutableTree.GetLastImmutable())

	candidatesS.Create([20]byte{1}, [20]byte{2}, [20]byte{3}, [32]byte{4}, 10, 0)
	candidatesS.SetOnline([32]byte{4})
	candidatesS.SetStakes([32]byte{4}, []types.Stake{
		{
			Owner:    [20]byte{1},
			Coin:     0,
			Value:    "1000000000000000000000",
			BipValue: "1000000000000000000000",
		},
	}, nil)
	candidatesS.RecalculateStakes(0)
	validators.SetNewValidators(candidatesS.GetNewCandidates(1))

	hash, version, err := mutableTree.Commit(validators)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "1D50F5F03FAB5D800DBF8D9254DDC68AEAC589BD30F2839A3A5B68887CE0E34C" {
		t.Fatalf("hash %X", hash)
	}

	state := new(types.AppState)
	validators.Export(state)

	bytes, err := json.Marshal(state.Validators)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "[{\"total_bip_stake\":\"1000000000000000000000\",\"public_key\":\"Mp0400000000000000000000000000000000000000000000000000000000000000\",\"accum_reward\":\"100\",\"absent_times\":\"________________________\"}]" {
		t.Log(string(bytes))
		t.Fatal("not equal JSON")
	}
}
