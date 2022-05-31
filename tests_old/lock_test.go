package tests_old

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
)

func TestLock_Simple(t *testing.T) {
	state := DefaultAppState() // generate default state

	state.FrozenFunds = []types.FrozenFund{
		{
			Height:  11,
			Address: types.Address{123},
			Coin:    0,
			Value:   "10000000000000099",
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

	t.Logf("%#v", app.GetEventsDB().LoadEvents(11)[0])

	{
		balance := app.CurrentState().Accounts().GetBalance(types.Address{123}, types.GetBaseCoinID())
		if balance.String() != "10000000000000099" {
			t.Fatalf("Recipient balance is not correct. Expected %s, got %s", "10000000000000099", balance)
		}
	}
}

func TestLockTx(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	stake := helpers.BipToPip(big.NewInt(10_000)).String()

	state.Validators = []types.Validator{
		{
			TotalBipStake: stake,
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
			Height: 5,
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

	// add address to genesis state
	value := helpers.StringToBigInt("100000000000000000000")
	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("100100000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state, 1) // create application

	//SendBeginBlock(app, 1) // send BeginBlock
	//SendEndBlock(app, 1)   // send EndBlock
	//SendCommit(app)        // send Commit

	SendBeginBlock(app, 2) // send BeginBlock
	SendEndBlock(app, 2)   // send EndBlock
	SendCommit(app)        // send Commit
	SendBeginBlock(app, 3) // send BeginBlock
	SendEndBlock(app, 3)   // send EndBlock
	SendCommit(app)        // send Commit
	SendBeginBlock(app, 4) // send BeginBlock
	SendEndBlock(app, 4)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 5) // send BeginBlock
	SendEndBlock(app, 5)   // send EndBlock
	SendCommit(app)        // send Commit
	SendBeginBlock(app, 6) // send BeginBlock
	SendEndBlock(app, 6)   // send EndBlock
	SendCommit(app)        // send Commit

	SendBeginBlock(app, 7) // send BeginBlock
	tx := CreateTx(app, address, transaction.TypeLock, transaction.LockData{
		Coin:     types.GetBaseCoinID(),
		DueBlock: 11,
		Value:    value,
	}, 0)

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	SendEndBlock(app, 7) // send EndBlock
	SendCommit(app)      // send Commit

	// check sender's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
		if balance.String() != "0" {
			t.Fatalf("Recipient balance is not correct. Expected %s, got %s", "0", balance)
		}
	}

	SendBeginBlock(app, 8)  // send BeginBlock
	SendEndBlock(app, 8)    // send EndBlock
	SendCommit(app)         // send Commit
	SendBeginBlock(app, 9)  // send BeginBlock
	SendEndBlock(app, 9)    // send EndBlock
	SendCommit(app)         // send Commit
	SendBeginBlock(app, 10) // send BeginBlock
	SendEndBlock(app, 10)   // send EndBlock
	SendCommit(app)         // send Commit
	SendBeginBlock(app, 11) // send BeginBlock
	SendEndBlock(app, 11)   // send EndBlock
	SendCommit(app)         // send Commit

	// check sender's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
		if balance.String() != "100000000000000000000" {
			t.Fatalf("Recipient balance is not correct. Expected %s, got %s", "100000000000000000000", balance)
		}
	}
}
