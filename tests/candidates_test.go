package tests

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
)

func TestRemoveCandidate(t *testing.T) {

	state := DefaultAppState() // generate default state
	state.Version = "v260"

	for i := byte(1); i < 255; i++ {

		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(i),
			Name:         "Test" + strconv.Itoa(int(i)),
			Symbol:       types.StrToCoinBaseSymbol("TEST" + strconv.Itoa(int(i))),
			Volume:       "10000000000000000000000000",
			Crr:          50,
			Reserve:      "10000000000000000000000000",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{255},
			Mintable:     false,
			Burnable:     false,
		})
		stake := helpers.BipToPip(big.NewInt(10000))
		if i <= 60 {
			state.Validators = append(state.Validators,
				types.Validator{
					TotalBipStake: stake.String(),
					PubKey:        types.Pubkey{i},
					AccumReward:   "10",
					AbsentTimes:   types.NewBitArray(24),
				},
			)
		} else {
			stake.Sub(stake, big.NewInt(100)).String()
		}

		state.Candidates = append(state.Candidates,
			types.Candidate{
				ID:             uint64(i),
				RewardAddress:  types.Address{i},
				OwnerAddress:   types.Address{i},
				ControlAddress: types.Address{i},
				TotalBipStake:  stake.String(),
				PubKey:         types.Pubkey{i},
				Commission:     10,
				Stakes: []types.Stake{
					{
						Owner:    types.Address{},
						Coin:     uint64(0),
						Value:    stake.String(),
						BipValue: stake.String(),
					},
				},
				Updates: []types.Stake{
					{
						Owner:    types.Address{i},
						Coin:     uint64(i),
						Value:    "10000000000000000000000000",
						BipValue: "0",
					},
				},
				Status: 2,
			})
	}

	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 3)
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	if len(appState.Candidates) != 100 {
		t.Errorf("want 100 active candidates (100 new top stakes), got %d", len(appState.Candidates))
	}
	if len(appState.Validators) != 64 {
		t.Errorf("want 64, got %d", len(appState.Validators))
	}
	if len(appState.FrozenFunds) != 308 { // (254 - 60 - 40) * 2
		t.Errorf("want 308, got %d", len(appState.FrozenFunds))
	}
}

func TestRemoveCandidate1(t *testing.T) {

	state := DefaultAppState() // generate default state
	state.Version = "v260"

	stake := helpers.BipToPip(big.NewInt(10000)).String()

	for i := byte(1); i < 255; i++ {
		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(i),
			Name:         "Test" + strconv.Itoa(int(i)),
			Symbol:       types.StrToCoinBaseSymbol("TEST" + strconv.Itoa(int(i))),
			Volume:       "10000000000000000000000000",
			Crr:          50,
			Reserve:      "10000000000000000000000000",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{255},
			Mintable:     false,
			Burnable:     false,
		})
		if i <= 60 {
			state.Validators = append(state.Validators,
				types.Validator{
					TotalBipStake: stake,
					PubKey:        types.Pubkey{i},
					AccumReward:   "10",
					AbsentTimes:   types.NewBitArray(24),
				},
			)
		}

		state.Candidates = append(state.Candidates,
			types.Candidate{
				ID:             uint64(i),
				RewardAddress:  types.Address{i},
				OwnerAddress:   types.Address{i},
				ControlAddress: types.Address{i},
				TotalBipStake:  stake,
				PubKey:         types.Pubkey{i},
				Commission:     10,
				Stakes: []types.Stake{
					{
						Owner:    types.Address{},
						Coin:     uint64(0),
						Value:    stake,
						BipValue: stake,
					},
				},
				Updates: []types.Stake{
					{
						Owner:    types.Address{i},
						Coin:     uint64(i),
						Value:    "10000000000000000000000000",
						BipValue: "0",
					},
				},
				Status: 2,
			})
	}

	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 3)
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	if len(appState.Candidates) != 100 {
		t.Errorf("want 160 active candidates (60 old validators and new 40 top stakes), got %d", len(appState.Candidates))
	}
	if len(appState.Validators) != 64 {
		t.Errorf("want 64, got %d", len(appState.Validators))
	}
	if len(appState.FrozenFunds) != 308 { // (254 - 100) * 2
		t.Errorf("want 188, got %d", len(appState.FrozenFunds))
	}
}

func TestRemoveCandidate2(t *testing.T) {

	state := DefaultAppState() // generate default state
	state.Version = "v260"

	stake := helpers.BipToPip(big.NewInt(10000)).String()

	for i := byte(1); i < 100; i++ {
		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(i),
			Name:         "Test" + strconv.Itoa(int(i)),
			Symbol:       types.StrToCoinBaseSymbol("TEST" + strconv.Itoa(int(i))),
			Volume:       "10000000000000000000000000",
			Crr:          50,
			Reserve:      "10000000000000000000000000",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{255},
			Mintable:     false,
			Burnable:     false,
		})
		if i <= 60 {
			state.Validators = append(state.Validators,
				types.Validator{
					TotalBipStake: stake,
					PubKey:        types.Pubkey{i},
					AccumReward:   "10",
					AbsentTimes:   types.NewBitArray(24),
				},
			)
		}

		state.Candidates = append(state.Candidates,
			types.Candidate{
				ID:             uint64(i),
				RewardAddress:  types.Address{i},
				OwnerAddress:   types.Address{i},
				ControlAddress: types.Address{i},
				TotalBipStake:  stake,
				PubKey:         types.Pubkey{i},
				Commission:     10,
				Stakes: []types.Stake{
					{
						Owner:    types.Address{},
						Coin:     uint64(0),
						Value:    stake,
						BipValue: stake,
					},
				},
				Updates: []types.Stake{
					{
						Owner:    types.Address{i},
						Coin:     uint64(i),
						Value:    "10000000000000000000000000",
						BipValue: "0",
					},
				},
				Status: 2,
			})
	}

	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 3)
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	if len(appState.Candidates) != 99 {
		t.Errorf("want 99 active candidates, got %d", len(appState.Candidates))
	}
	if len(appState.Validators) != 64 {
		t.Errorf("want 64, got %d", len(appState.Validators))
	}
	if len(appState.FrozenFunds) != 0 {
		t.Errorf("want 0, got %d", len(appState.FrozenFunds))
	}
}

func TestRemoveCandidate4(t *testing.T) {

	state := DefaultAppState() // generate default state
	state.Version = "v260"

	stake := helpers.BipToPip(big.NewInt(10000)).String()

	for i := byte(1); i < 101; i++ {
		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(i),
			Name:         "Test" + strconv.Itoa(int(i)),
			Symbol:       types.StrToCoinBaseSymbol("TEST" + strconv.Itoa(int(i))),
			Volume:       "10000000000000000000000000",
			Crr:          50,
			Reserve:      "10000000000000000000000000",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{255},
			Mintable:     false,
			Burnable:     false,
		})
		if i <= 60 {
			state.Validators = append(state.Validators,
				types.Validator{
					TotalBipStake: stake,
					PubKey:        types.Pubkey{i},
					AccumReward:   "10",
					AbsentTimes:   types.NewBitArray(24),
				},
			)
		}

		state.Candidates = append(state.Candidates,
			types.Candidate{
				ID:             uint64(i),
				RewardAddress:  types.Address{i},
				OwnerAddress:   types.Address{i},
				ControlAddress: types.Address{i},
				TotalBipStake:  stake,
				PubKey:         types.Pubkey{i},
				Commission:     10,
				Stakes: []types.Stake{
					{
						Owner:    types.Address{},
						Coin:     uint64(0),
						Value:    stake,
						BipValue: stake,
					},
				},
				Updates: []types.Stake{
					{
						Owner:    types.Address{i},
						Coin:     uint64(i),
						Value:    "10000000000000000000000000",
						BipValue: "0",
					},
				},
				Status: 2,
			})
	}

	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 3)
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	if len(appState.Candidates) != 100 {
		t.Errorf("want 100 active candidates, got %d", len(appState.Candidates))
	}
	if len(appState.Validators) != 64 {
		t.Errorf("want 64, got %d", len(appState.Validators))
	}
	if len(appState.FrozenFunds) != 0 {
		t.Errorf("want 0, got %d", len(appState.FrozenFunds))
	}
}

func TestRemoveCandidate5(t *testing.T) {

	state := DefaultAppState() // generate default state
	state.Version = "v260"

	stake := helpers.BipToPip(big.NewInt(10000)).String()

	for i := byte(1); i < 102; i++ {
		state.Coins = append(state.Coins, types.Coin{
			ID:           uint64(i),
			Name:         "Test" + strconv.Itoa(int(i)),
			Symbol:       types.StrToCoinBaseSymbol("TEST" + strconv.Itoa(int(i))),
			Volume:       "10000000000000000000000000",
			Crr:          50,
			Reserve:      "10000000000000000000000000",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{255},
			Mintable:     false,
			Burnable:     false,
		})
		if i <= 60 {
			state.Validators = append(state.Validators,
				types.Validator{
					TotalBipStake: stake,
					PubKey:        types.Pubkey{i},
					AccumReward:   "10",
					AbsentTimes:   types.NewBitArray(24),
				},
			)
		}

		state.Candidates = append(state.Candidates,
			types.Candidate{
				ID:             uint64(i),
				RewardAddress:  types.Address{i},
				OwnerAddress:   types.Address{i},
				ControlAddress: types.Address{i},
				TotalBipStake:  stake,
				PubKey:         types.Pubkey{i},
				Commission:     10,
				Stakes: []types.Stake{
					{
						Owner:    types.Address{},
						Coin:     uint64(0),
						Value:    stake,
						BipValue: stake,
					},
				},
				Updates: []types.Stake{
					{
						Owner:    types.Address{i},
						Coin:     uint64(i),
						Value:    "10000000000000000000000000",
						BipValue: "0",
					},
				},
				Status: 2,
			})
	}

	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 3)
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	appState := app.CurrentState().Export()
	if err := appState.Verify(); err != nil {
		t.Fatalf("export err: %v", err)
	}
	if len(appState.Candidates) != 100 {
		t.Errorf("want 100 active candidates, got %d", len(appState.Candidates))
	}
	if len(appState.Validators) != 64 {
		t.Errorf("want 64, got %d", len(appState.Validators))
	}
	if len(appState.FrozenFunds) != 2 {
		t.Errorf("want 2, got %d", len(appState.FrozenFunds))
	}
}
