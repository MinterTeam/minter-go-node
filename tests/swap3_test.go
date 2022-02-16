package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
)

func TestSwap3_SimpleGas0(t *testing.T) {
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

	address, pk := CreateAddress() // create account for test
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
		{
			ID:           1,
			Name:         "Test 1",
			Symbol:       types.StrToCoinBaseSymbol("TEST1"),
			Volume:       "100004099999999999999999",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &address,
			Mintable:     false,
			Burnable:     false,
		},
	}

	state.Pools = []types.Pool{
		{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "3500000000000000000000000000",
			Reserve1: "10000000000000000000000000",
			ID:       2,
			Orders:   nil,
		},
	}
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    0,
		Reserve0: "100000000000000000000000",
		Reserve1: "10000000000000000000000000",
		ID:       1,
		Orders: []types.Order{{
			IsSale:  true,
			Volume0: "10000000000000000000",    // want to buy
			Volume1: "99998528242522874800000", // want to sell
			ID:      1,
			Owner:   types.Address{},
			Height:  300,
		}},
	})
	state.Commission.Coin = 0
	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("997987309832970547704").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("4099999999999999999").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

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

	SendBeginBlock(app, 12) // send BeginBlock
	{
		tx := CreateTx(app, address, transaction.TypeBuySwapPool, transaction.BuySwapPoolDataV260{
			Coins:              []types.CoinID{1, 0},
			ValueToBuy:         helpers.StringToBigInt("39919492393318821938180"),
			MaximumValueToSell: helpers.StringToBigInt("4000000000000000000"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code == code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
		//for _, event := range response.Events {
		//	for _, tag := range event.Attributes {
		//		t.Log(tag.String())
		//	}
		//}
	}
	{
		tx := CreateTx(app, address, transaction.TypeBuySwapPool, transaction.BuySwapPoolDataV260{
			Coins:              []types.CoinID{1, 0},
			ValueToBuy:         helpers.StringToBigInt("39919492393318821938180"),
			MaximumValueToSell: helpers.StringToBigInt("4004004004004004004"),
		}, 0)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
		for _, event := range response.Events {
			for _, tag := range event.Attributes {
				t.Log(tag.String())
			}
		}
	}
	SendEndBlock(app, 12) // send EndBlock
	SendCommit(app)       // send Commit
}
func TestSwap3_SimpleGas1(t *testing.T) {
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

	address, pk := CreateAddress() // create account for test
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
		{
			ID:           1,
			Name:         "Test 1",
			Symbol:       types.StrToCoinBaseSymbol("TEST1"),
			Volume:       "100004099999999999999999",
			Crr:          0,
			Reserve:      "0",
			MaxSupply:    "90000000000000000000000000000",
			Version:      0,
			OwnerAddress: &address,
			Mintable:     false,
			Burnable:     false,
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
	state.NextOrderID = 2
	state.Pools = append(state.Pools, types.Pool{
		Coin0:    1,
		Coin1:    0,
		Reserve0: "100000000000000000000000",
		Reserve1: "10000000000000000000000000",
		ID:       1,
		Orders: []types.Order{{
			IsSale:  true,
			Volume0: "10000000000000000000",    // want to buy
			Volume1: "99998528242522874800000", // want to sell
			ID:      1,
			Owner:   types.Address{},
			Height:  30,
		}},
	})
	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("997987309832970547704").String(),
			},
			{
				Coin:  1,
				Value: helpers.StringToBigInt("4099999999999999999").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

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

	SendBeginBlock(app, 12) // send BeginBlock
	{
		tx := CreateTx(app, address, transaction.TypeBuySwapPool, transaction.BuySwapPoolDataV260{
			Coins:              []types.CoinID{1, 0},
			ValueToBuy:         helpers.StringToBigInt("39919492393318821938180"),
			MaximumValueToSell: helpers.StringToBigInt("4004004004004004004"),
		}, 1)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code == code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}

	}
	{
		tx := CreateTx(app, address, transaction.TypeBuySwapPool, transaction.BuySwapPoolDataV260{
			Coins:              []types.CoinID{1, 0},
			ValueToBuy:         helpers.StringToBigInt("39919492393318821938180"),
			MaximumValueToSell: helpers.StringToBigInt("4004004004004004005"),
		}, 1)

		response := SendTx(app, SignTx(pk, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
		for _, event := range response.Events {
			for _, tag := range event.Attributes {
				t.Log(tag.String())
			}
		}
	}
	SendEndBlock(app, 12) // send EndBlock
	SendCommit(app)       // send Commit
}
