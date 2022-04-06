package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
)

func TestStakeMove_Simple(t *testing.T) {
	state := DefaultAppState() // generate default state

	stake := helpers.BipToPip(big.NewInt(10_000)).String()

	state.Validators = []types.Validator{
		{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{1},
			AccumReward:   "1000000",
			AbsentTimes:   nil,
		},
	}

	state.Candidates = []types.Candidate{
		{
			ID:             1,
			RewardAddress:  types.Address{1},
			OwnerAddress:   types.Address{1},
			ControlAddress: types.Address{1},
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{1},
			Commission:     5,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{5},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates:                  nil,
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		},
	}
	//state.Version = "v300"

	state.FrozenFunds = []types.FrozenFund{
		{
			Height:            11,
			Address:           types.Address{123},
			CandidateKey:      &types.Pubkey{123},
			CandidateID:       123,
			Coin:              0,
			Value:             "10000000000000099",
			MoveToCandidateID: 1,
		},
	}

	app := CreateApp(state) // create application

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 11) // send BeginBlock
	SendEndBlock(app, 11)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 12) // send BeginBlock
	SendEndBlock(app, 12)   // send EndBlock
	SendCommit(app)         // send Commit

	t.Log(app.GetEventsDB().LoadEvents(11)[0])
}
