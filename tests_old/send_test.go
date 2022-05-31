package tests_old

import (
	"math/big"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
)

func TestSend(t *testing.T) {
	address, pk := CreateAddress() // create account for test

	state := DefaultAppState() // generate default state

	// add address to genesis state
	state.Accounts = append(state.Accounts, types.Account{
		Address: address,
		Balance: []types.Balance{
			{
				Coin:  uint64(types.GetBaseCoinID()),
				Value: helpers.StringToBigInt("100000000000000000000").String(),
			},
		},
		Nonce:        0,
		MultisigData: nil,
	})

	app := CreateApp(state) // create application
	SendBeginBlock(app, 1)  // send BeginBlock

	recipient, _ := CreateAddress() // generate recipient
	value := big.NewInt(1)

	tx := CreateTx(app, address, transaction.TypeSend, transaction.SendData{
		Coin:  types.GetBaseCoinID(),
		To:    recipient,
		Value: value,
	}, 0)

	response := SendTx(app, SignTx(pk, tx)) // compose and send tx

	// check that result is OK
	if response.Code != code.OK {
		t.Fatalf("Response code is not OK: %s, %d", response.Log, response.Code)
	}

	SendEndBlock(app, 1) // send EndBlock
	SendCommit(app)      // send Commit

	// check recipient's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(recipient, types.GetBaseCoinID())
		if balance.Cmp(value) != 0 {
			t.Fatalf("Recipient balance is not correct. Expected %s, got %s", value, balance)
		}
	}

	// check sender's balance
	{
		balance := app.CurrentState().Accounts().GetBalance(address, types.GetBaseCoinID())
		if balance.String() != "99989999999999999999" {
			t.Fatalf("Recipient balance is not correct. Expected %s, got %s", "98999999999999999999", balance)
		}
	}
}
