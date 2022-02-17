package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
	"time"
)

func TestReward_Simple(t *testing.T) {
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
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 10,
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

	app := CreateApp(state) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 11) // send BeginBlock
	SendEndBlock(app, 11)   // send EndBlock
	SendCommit(app)         // send Commit

	t.Log(app.UpdateVersions()[1])
	t.Log(app.GetEventsDB().LoadEvents(11)[0])
	t.Log(app.CurrentState().App().Reward())
}

func TestReward_Update_Up(t *testing.T) {
	state := DefaultAppState() // generate default state

	address, pk := CreateAddress() // create account for test

	state.Accounts = []types.Account{
		{
			Address: address,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("1000000000100000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
	}
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
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 10,
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

	app := CreateApp(state) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	{
		SendBeginBlock(app, 11) // send BeginBlock

		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
			Coins:             []types.CoinID{0, types.USDTID},
			ValueToSell:       helpers.StringToBigInt("1000000000000000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

		SendEndBlock(app, 11) // send EndBlock
		SendCommit(app)       // send Commit
	}

	SendBeginBlock(app, 12) // send BeginBlock
	SendEndBlock(app, 12)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 13, time.Unix(1643208154, 0)) // send BeginBlock
	SendEndBlock(app, 13)                             // send EndBlock
	SendCommit(app)                                   // send Commit

	t.Log(app.GetEventsDB().LoadEvents(11)[0])
	t.Log(app.GetEventsDB().LoadEvents(13)[0])
}
func TestReward_Update_Down(t *testing.T) {
	state := DefaultAppState() // generate default state

	address, pk := CreateAddress() // create account for test

	state.Accounts = []types.Account{
		{
			Address: address,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("1000000000100000000000000000").String(),
				},
				{
					Coin:  uint64(types.USDTID),
					Value: helpers.StringToBigInt("10000000000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
	}
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
	state.UpdateVotes = []types.UpdateVote{
		{
			Height: 10,
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
			Volume:       "20000000000000000000000000",
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

	app := CreateApp(state) // create application

	SendBeginBlock(app, 9) // send BeginBlock
	SendEndBlock(app, 9)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit

	{
		SendBeginBlock(app, 11) // send BeginBlock

		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
			Coins:             []types.CoinID{types.USDTID, 0},
			ValueToSell:       helpers.StringToBigInt("1000000000000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

		SendEndBlock(app, 11) // send EndBlock
		SendCommit(app)       // send Commit
	}

	SendBeginBlock(app, 12) // send BeginBlock
	SendEndBlock(app, 12)   // send EndBlock
	SendCommit(app)         // send Commit

	SendBeginBlock(app, 13, time.Unix(1643208154, 0)) // send BeginBlock
	SendEndBlock(app, 13)                             // send EndBlock
	SendCommit(app)                                   // send Commit

	t.Log(app.GetEventsDB().LoadEvents(11)[0])
	t.Log(app.GetEventsDB().LoadEvents(13)[0])

	{
		SendBeginBlock(app, 14) // send BeginBlock

		tx := CreateTx(app, address, transaction.TypeSellSwapPool, transaction.SellSwapPoolDataV230{
			Coins:             []types.CoinID{types.USDTID, 0},
			ValueToSell:       helpers.StringToBigInt("5000000000000000000000000"),
			MinimumValueToBuy: helpers.StringToBigInt("1"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

		SendEndBlock(app, 14) // send EndBlock
		SendCommit(app)       // send Commit
	}

	SendBeginBlock(app, 15, time.Unix(1643640154, 0)) // send BeginBlock
	SendEndBlock(app, 15)                             // send EndBlock
	SendCommit(app)                                   // send Commit

	t.Log(app.GetEventsDB().LoadEvents(15)[0])
}
