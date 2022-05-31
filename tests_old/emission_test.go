package tests_old

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
)

func TestLock_Emission(t *testing.T) {
	state := DefaultAppState() // generate default state

	stake := helpers.BipToPip(big.NewInt(10_000))
	state.Validators = []types.Validator{
		{
			TotalBipStake: big.NewInt(0).Add(stake, stake).String(),
			PubKey:        types.Pubkey{1},
			AccumReward:   "1000000",
			AbsentTimes:   types.NewBitArray(24),
		},
	}
	state.Candidates = []types.Candidate{
		{
			ID:             1,
			RewardAddress:  types.Address{1},
			OwnerAddress:   types.Address{1},
			ControlAddress: types.Address{1},
			TotalBipStake:  big.NewInt(0).Add(stake, stake).String(),
			PubKey:         types.Pubkey{1},
			Commission:     99,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{5},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
				{
					Owner:    types.Address{11},
					Coin:     0,
					Value:    stake.String(),
					BipValue: stake.String(),
				},
			},
			Updates:                  nil,
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		},
	}
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 43702611 - 9150000 - 3,
			Votes: []types.Pubkey{
				[32]byte{1},
			},
			Version: "v300",
		},
	}
	state.Coins = []types.Coin{
		{
			ID:           types.USDTID,
			Name:         "USDT (Tether USD, Ethereum)",
			Symbol:       types.StrToCoinBaseSymbol("USDTE"),
			Volume:       "10000000000000000000000000",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "10000000000000000000000000",
			Version:      0,
			OwnerAddress: &types.Address{},
			Mintable:     true,
			Burnable:     true,
		},
	}
	state.Pools = []types.Pool{
		{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "3500000000000000000000000000",
			Reserve1: "10000000000000000000000000",
			ID:       1,
			Orders:   nil,
		},
	}

	initialHeight := int64(43702611 - 9150000 - 5)
	app := CreateApp(state, initialHeight) // create application

	SendBeginBlock(app, initialHeight) // send BeginBlock
	SendEndBlock(app, initialHeight)   // send EndBlock
	SendCommit(app)                    // send Commit

	if _, ok := app.CurrentState().Swap().(*swap.SwapV2); !ok {
		t.Error("swap required v2")
	}

	SendBeginBlock(app, initialHeight+1) // send BeginBlock
	SendEndBlock(app, initialHeight+1)   // send EndBlock
	SendCommit(app)                      // send Commit

	SendBeginBlock(app, initialHeight+2) // send BeginBlock
	SendEndBlock(app, initialHeight+2)   // send EndBlock
	SendCommit(app)                      // send Commit

	SendBeginBlock(app, initialHeight+3) // send BeginBlock
	SendEndBlock(app, initialHeight+3)   // send EndBlock
	SendCommit(app)                      // send Commit

	SendBeginBlock(app, initialHeight+4) // send BeginBlock
	SendEndBlock(app, initialHeight+4)   // send EndBlock
	SendCommit(app)                      // send Commit

	t.Logf("%#v", app.GetEventsDB().LoadEvents(uint32(initialHeight + 4))[0])
	t.Log(app.CurrentState().App().Reward())
	t.Log(app.GetEmission())

	SendBeginBlock(app, initialHeight+5) // send BeginBlock
	SendEndBlock(app, initialHeight+5)   // send EndBlock
	SendCommit(app)                      // send Commit

	SendBeginBlock(app, initialHeight+6) // send BeginBlock
	SendEndBlock(app, initialHeight+6)   // send EndBlock
	SendCommit(app)                      // send Commit

	t.Logf("%#v", app.GetEventsDB().LoadEvents(uint32(initialHeight + 6))[0])
	t.Log(app.CurrentState().App().Reward())
	t.Log(app.GetEmission())

	SendBeginBlock(app, initialHeight+7) // send BeginBlock
	SendEndBlock(app, initialHeight+7)   // send EndBlock
	SendCommit(app)                      // send Commit

	SendBeginBlock(app, initialHeight+8) // send BeginBlock
	SendEndBlock(app, initialHeight+8)   // send EndBlock
	SendCommit(app)                      // send Commit

	t.Logf("%#v", app.GetEventsDB().LoadEvents(uint32(initialHeight+8)))
	t.Log(app.CurrentState().App().Reward())
	t.Log(app.GetEmission())

	if _, ok := app.CurrentState().Swap().(*swap.SwapV2); !ok {
		t.Error("swap required v2")
	}
}
