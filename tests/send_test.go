package tests

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math"
	"math/big"
	"testing"
)

type diffBalance struct {
	Coin  types.CoinID
	Value *big.Int
}

func TestEditBalance(t *testing.T) {
	sender := CreateAddress() // create account for test

	helper := NewHelper(DefaultAppState(sender.address))

	var transactions []func() (transaction.Transaction, []*diffBalance)

	for _, coin := range []types.CoinID{types.BasecoinID, types.USDTID} {
		coin := coin
		{
			recipient := CreateAddress()
			value := big.NewInt(1)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance) {
				return helper.CreateTx(sender.privateKey, transaction.SendData{
						Coin:  coin,
						To:    recipient.address,
						Value: value,
					}, types.USDTID), []*diffBalance{{
						Coin:  coin,
						Value: value}}
			})
		}
		{
			value := big.NewInt(1)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance) {
				return helper.CreateTx(sender.privateKey, transaction.LockData{
						DueBlock: math.MaxUint32,
						Coin:     coin,
						Value:    value,
					}, types.USDTID), []*diffBalance{{
						Coin:  coin,
						Value: value}}
			})
		}
	}

	for _, txFunc := range transactions {
		tx, diffs := txFunc()
		t.Run(fmt.Sprintf("TxType=%s,%#v", tx.Type, tx.GetDecodedData()), func(t *testing.T) {
			testBalance(t, helper, tx, diffs)
		})
	}
}
func testBalance(t *testing.T, helper *Helper, tx transaction.Transaction, diffs []*diffBalance) {
	sender, _ := tx.Sender()

	initialCommissionCoinBalance := helper.app.CurrentState().Accounts().GetBalance(sender, tx.CommissionCoin())

	var initialBalances []*big.Int
	for _, diff := range diffs {
		initialBalances = append(initialBalances, helper.app.CurrentState().Accounts().GetBalance(sender, diff.Coin))
	}

	_, results := helper.NextBlock(tx)

	for _, resp := range results {
		if resp.Code != code.OK {
			t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
		}
	}

	var checkedCommissionCoin bool
	for i, diff := range diffs {
		newBalance := big.NewInt(0).Sub(initialBalances[i], diff.Value)
		if diff.Coin == tx.CommissionCoin() {
			if tx.CommissionCoin() != types.USDTID {
				panic("unimplemented test commission coin not equal USDT")
			}
			checkedCommissionCoin = true
			newBalance.Sub(newBalance, tx.MulGasPrice(tx.Price(helper.app.CurrentState().Commission().GetCommissions())))
		}
		if newBalance.String() != helper.app.CurrentState().Accounts().GetBalance(sender, diff.Coin).String() {
			t.Errorf("error sender CoinID-%d balance", diff.Coin)
		}
	}

	if !checkedCommissionCoin {
		if big.NewInt(0).Sub(initialCommissionCoinBalance, tx.Price(helper.app.CurrentState().Commission().GetCommissions())).String() != helper.app.CurrentState().Accounts().GetBalance(sender, tx.CommissionCoin()).String() {
			t.Error("error sender usd balance")
		}
	}

}

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
