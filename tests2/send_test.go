package tests

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"testing"
)

func TestSend(t *testing.T) {
	sender := CreateAddress() // create account for test

	helper := NewHelper(DefaultAppState(sender.address))

	initialBIPBalance := helper.app.CurrentState().Accounts().GetBalance(sender.address, 0)
	initialUSDTBalance := helper.app.CurrentState().Accounts().GetBalance(sender.address, types.USDTID)

	recipient := CreateAddress() // generate recipient
	value := big.NewInt(1)
	tx := helper.CreateTx(sender.privateKey, transaction.SendData{
		Coin:  types.GetBaseCoinID(),
		To:    recipient.address,
		Value: value,
	}, types.USDTID)

	_, results := helper.NextBlock(tx)

	for _, resp := range results {
		if resp.Code != code.OK {
			t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
		}
	}

	if big.NewInt(0).Sub(initialUSDTBalance, helper.app.CurrentState().Commission().GetCommissions().Send).String() != helper.app.CurrentState().Accounts().GetBalance(sender.address, types.USDTID).String() {
		t.Fatalf("error sender usd balance")
	}
	if big.NewInt(0).Sub(initialBIPBalance, value).String() != helper.app.CurrentState().Accounts().GetBalance(sender.address, 0).String() {
		t.Fatalf("error sender bip balance")
	}
	if value.String() != helper.app.CurrentState().Accounts().GetBalance(recipient.address, 0).String() {
		t.Fatalf("error recipient bip balance")
	}
}
