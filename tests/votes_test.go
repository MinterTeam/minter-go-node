package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"testing"
)

func TestVoteCommissionFail(t *testing.T) {
	privateKey1, _ := crypto.GenerateKey() // create accounts for test
	address1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey() // create accounts for test
	address2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey() // create accounts for test
	address3 := crypto.PubkeyToAddress(privateKey3.PublicKey)
	privateKey4, _ := crypto.GenerateKey() // create accounts for test
	address4 := crypto.PubkeyToAddress(privateKey4.PublicKey)
	privateKey5, _ := crypto.GenerateKey() // create accounts for test
	address5 := crypto.PubkeyToAddress(privateKey5.PublicKey)
	privateKey6, _ := crypto.GenerateKey() // create accounts for test
	address6 := crypto.PubkeyToAddress(privateKey6.PublicKey)

	state := DefaultAppState() // generate default state

	// add address to genesis state
	state.Accounts = append(state.Accounts,
		types.Account{
			Address: address1,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address2,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address3,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address4,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address5,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address6,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
	)
	stake := helpers.BipToPip(big.NewInt(10000)).String()
	state.Validators = append(state.Validators,
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{1},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{2},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{3},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{4},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{5},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{6},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
	)
	state.Candidates = append(state.Candidates,
		types.Candidate{
			ID:             1,
			RewardAddress:  address1,
			OwnerAddress:   address1,
			ControlAddress: address1,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{1},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             2,
			RewardAddress:  address2,
			OwnerAddress:   address2,
			ControlAddress: address2,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{2},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             3,
			RewardAddress:  address3,
			OwnerAddress:   address3,
			ControlAddress: address3,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{3},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             4,
			RewardAddress:  address4,
			OwnerAddress:   address4,
			ControlAddress: address4,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{4},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             5,
			RewardAddress:  address5,
			OwnerAddress:   address5,
			ControlAddress: address5,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{5},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             6,
			RewardAddress:  address6,
			OwnerAddress:   address6,
			ControlAddress: address6,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{6},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
	)
	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 3) // send BeginBlock
	{
		tx := CreateTx(app, address1, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{1},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey1, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address2, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{2},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey2, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address3, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{3},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey3, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 4) // send BeginBlock

	{
		tx := CreateTx(app, address5, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{5},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey5, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address6, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{6},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey6, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}

	{ // Diff
		tx := CreateTx(app, address4, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{4},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(2e18), // Diff
		})

		response := SendTx(app, SignTx(privateKey4, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	SendEndBlock(app, 4) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 5)
	SendEndBlock(app, 5) // send EndBlock
	SendCommit(app)      // send Commit

	commissions := app.CurrentState().Commission().GetCommissions()
	if commissions.Send.Cmp(big.NewInt(9999)) == 0 {
		t.Fatalf("comission send want uncorrect. Expected %s, got %s", big.NewInt(2e18), commissions.Send)
	}
}

func TestVoteCommissionOKUpdateVersion(t *testing.T) {
	privateKey1, _ := crypto.GenerateKey() // create accounts for test
	address1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey() // create accounts for test
	address2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey() // create accounts for test
	address3 := crypto.PubkeyToAddress(privateKey3.PublicKey)
	privateKey4, _ := crypto.GenerateKey() // create accounts for test
	address4 := crypto.PubkeyToAddress(privateKey4.PublicKey)
	privateKey5, _ := crypto.GenerateKey() // create accounts for test
	address5 := crypto.PubkeyToAddress(privateKey5.PublicKey)
	privateKey6, _ := crypto.GenerateKey() // create accounts for test
	address6 := crypto.PubkeyToAddress(privateKey6.PublicKey)

	state := DefaultAppState() // generate default state

	// add address to genesis state
	state.Accounts = append(state.Accounts,
		types.Account{
			Address: address1,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address2,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address3,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address4,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address5,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
		types.Account{
			Address: address6,
			Balance: []types.Balance{
				{
					Coin:  uint64(types.GetBaseCoinID()),
					Value: helpers.StringToBigInt("100000000000000000000").String(),
				},
			},
			Nonce:        0,
			MultisigData: nil,
		},
	)
	stake := helpers.BipToPip(big.NewInt(10000)).String()
	state.Validators = append(state.Validators,
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{1},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{2},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{3},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{4},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{5},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
		types.Validator{
			TotalBipStake: stake,
			PubKey:        types.Pubkey{6},
			AccumReward:   "10",
			AbsentTimes:   nil,
		},
	)
	state.Candidates = append(state.Candidates,
		types.Candidate{
			ID:             1,
			RewardAddress:  address1,
			OwnerAddress:   address1,
			ControlAddress: address1,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{1},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             2,
			RewardAddress:  address2,
			OwnerAddress:   address2,
			ControlAddress: address2,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{2},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             3,
			RewardAddress:  address3,
			OwnerAddress:   address3,
			ControlAddress: address3,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{3},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             4,
			RewardAddress:  address4,
			OwnerAddress:   address4,
			ControlAddress: address4,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{4},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             5,
			RewardAddress:  address5,
			OwnerAddress:   address5,
			ControlAddress: address5,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{5},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
		types.Candidate{
			ID:             6,
			RewardAddress:  address6,
			OwnerAddress:   address6,
			ControlAddress: address6,
			TotalBipStake:  stake,
			PubKey:         types.Pubkey{6},
			Commission:     10,
			Stakes: []types.Stake{
				{
					Owner:    types.Address{},
					Coin:     0,
					Value:    stake,
					BipValue: stake,
				},
			},
			Updates: nil,
			Status:  2,
		},
	)
	app := CreateApp(state) // create application

	SendBeginBlock(app, 1)
	{
		tx := CreateTx(app, address1, transaction.TypeVoteUpdate, transaction.VoteUpdateData{
			PubKey:  types.Pubkey{1},
			Height:  2,
			Version: "a",
		})

		response := SendTx(app, SignTx(privateKey1, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address2, transaction.TypeVoteUpdate, transaction.VoteUpdateData{
			PubKey:  types.Pubkey{2},
			Height:  2,
			Version: "a",
		})

		response := SendTx(app, SignTx(privateKey2, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address3, transaction.TypeVoteUpdate, transaction.VoteUpdateData{
			PubKey:  types.Pubkey{3},
			Height:  2,
			Version: "a",
		})

		response := SendTx(app, SignTx(privateKey3, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address4, transaction.TypeVoteUpdate, transaction.VoteUpdateData{
			PubKey:  types.Pubkey{4},
			Height:  2,
			Version: "a",
		})

		response := SendTx(app, SignTx(privateKey4, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address5, transaction.TypeVoteUpdate, transaction.VoteUpdateData{
			PubKey:  types.Pubkey{5},
			Height:  2,
			Version: "aA",
		})

		response := SendTx(app, SignTx(privateKey5, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 2)
	SendEndBlock(app, 2) // send EndBlock
	SendCommit(app)      // send Commit

	if len(app.UpdateVersions()) != 1 {
		t.Fatalf("not updates")
	}

	SendBeginBlock(app, 3) // send BeginBlock
	{
		tx := CreateTx(app, address1, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{1},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey1, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address2, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{2},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey2, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address3, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{3},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey3, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	SendEndBlock(app, 3) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 4) // send BeginBlock

	{
		tx := CreateTx(app, address5, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{5},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey5, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	{
		tx := CreateTx(app, address6, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{6},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(9999),
		})

		response := SendTx(app, SignTx(privateKey6, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}

	{ // Diff
		tx := CreateTx(app, address4, transaction.TypeVoteCommission, transaction.VoteCommissionData{
			PubKey: types.Pubkey{4},
			Coin:   types.GetBaseCoinID(),
			Height: 5,
			Send:   big.NewInt(2e18), // Diff
		})

		response := SendTx(app, SignTx(privateKey4, tx)) // compose and send tx

		// check that result is OK
		if response.Code != code.OK {
			t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
		}
	}
	SendEndBlock(app, 4) // send EndBlock
	SendCommit(app)      // send Commit

	SendBeginBlock(app, 5)
	SendEndBlock(app, 5) // send EndBlock
	SendCommit(app)      // send Commit

	commissions := app.CurrentState().Commission().GetCommissions()
	if commissions.Send.Cmp(big.NewInt(9999)) != 0 {
		t.Fatalf("comission send is not correct. Expected %s, got %s", big.NewInt(9999), commissions.Send)
	}
}
