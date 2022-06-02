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
	sender := CreateAddress()

	helper := NewHelper(DefaultAppState(sender.address))

	// diffs == nil is wanted fail tx
	var transactions []func() (tx transaction.Transaction, diffs []*diffBalance, desc string)

	for _, coinCommission := range []types.CoinID{types.BasecoinID, types.USDTID} {
		coinCommission := coinCommission

		for _, coinAction := range []types.CoinID{types.BasecoinID, types.USDTID} {
			coinAction := coinAction
			{
				recipient := CreateAddress()
				value := big.NewInt(1)
				transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
					return helper.CreateTx(sender.privateKey, transaction.SendData{
							Coin:  coinAction,
							To:    recipient.address,
							Value: value,
						}, coinCommission), []*diffBalance{{
							Coin:  coinAction,
							Value: value}}, fmt.Sprintf("coinAction%dcoinCommission%d", coinAction, coinCommission)
				})
			}
			{
				value := big.NewInt(1)
				transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
					return helper.CreateTx(sender.privateKey, transaction.LockData{
							DueBlock: math.MaxUint32,
							Coin:     coinAction,
							Value:    value,
						}, coinCommission), []*diffBalance{{
							Coin:  coinAction,
							Value: value}}, fmt.Sprintf("coinAction%dcoinCommission%d", coinAction, coinCommission)
				})
			}
		}

		{
			value := big.NewInt(1e18)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.SellSwapPoolDataV260{
						Coins:             []types.CoinID{types.GetBaseCoinID(), types.USDTID},
						ValueToSell:       value,
						MinimumValueToBuy: big.NewInt(1),
					}, types.USDTID), []*diffBalance{{
						Coin:  types.GetBaseCoinID(),
						Value: value}, {
						Coin:  types.USDTID,
						Value: big.NewInt(0).Neg(big.NewInt(997001999999999)),
					}}, "sell0"
			})
		}
		{
			value := big.NewInt(1e15)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.SellSwapPoolDataV260{
						Coins:             []types.CoinID{types.USDTID, types.GetBaseCoinID()},
						ValueToSell:       value,
						MinimumValueToBuy: big.NewInt(1),
					}, types.USDTID), []*diffBalance{{
						Coin:  types.USDTID,
						Value: value}, {
						Coin:  types.GetBaseCoinID(),
						Value: big.NewInt(0).Neg(big.NewInt(997001999999998806)),
					}}, "sell1993"
			})
		}
		{
			value := big.NewInt(1e15)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.BuySwapPoolDataV260{
						Coins:              []types.CoinID{types.GetBaseCoinID(), types.USDTID},
						ValueToBuy:         value,
						MaximumValueToSell: big.NewInt(2e18),
					}, types.USDTID), []*diffBalance{{
						Coin:  types.GetBaseCoinID(),
						Value: big.NewInt(1003007015031063931)}, {
						Coin:  types.USDTID,
						Value: big.NewInt(0).Neg(value),
					}}, "buy1993"
			})
		}
		{
			value := big.NewInt(1e15)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.BuySwapPoolDataV260{
						Coins:              []types.CoinID{types.GetBaseCoinID(), types.USDTID},
						ValueToBuy:         value,
						MaximumValueToSell: big.NewInt(2e18),
					}, types.USDTID), []*diffBalance{{
						Coin:  types.GetBaseCoinID(),
						Value: big.NewInt(1003007015031063931)}, {
						Coin:  types.USDTID,
						Value: big.NewInt(0).Neg(value),
					}}, "buy1993"
			})
		}
		{
			value := big.NewInt(1e18)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.BuySwapPoolDataV260{
						Coins:              []types.CoinID{types.USDTID, types.GetBaseCoinID()},
						ValueToBuy:         value,
						MaximumValueToSell: big.NewInt(2e15),
					}, types.USDTID), []*diffBalance{{
						Coin:  types.USDTID,
						Value: big.NewInt(1003007015031066)}, {
						Coin:  types.GetBaseCoinID(),
						Value: big.NewInt(0).Neg(value),
					}}, "buy0"
			})
		}

		{
			value := big.NewInt(1)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.LockData{
					DueBlock: math.MaxUint32,
					Coin:     404,
					Value:    value,
				}, coinCommission), nil, fmt.Sprintf("coinAction%dcoinCommission%d", 404, coinCommission)
			})
		}
		{
			recipient := CreateAddress()
			value := big.NewInt(1)
			transactions = append(transactions, func() (transaction.Transaction, []*diffBalance, string) {
				return helper.CreateTx(sender.privateKey, transaction.SendData{
					Coin:  404,
					To:    recipient.address,
					Value: value,
				}, coinCommission), nil, fmt.Sprintf("coinAction%dcoinCommission%d", 404, coinCommission)
			})
		}
	}

	for _, txFunc := range transactions {
		tx, diffs, description := txFunc()
		t.Run("Type"+tx.Type.String()+""+description, func(t *testing.T) {
			testBalance(t, NewHelper(DefaultAppState(sender.address)), tx, diffs)
		})
	}
}

func testBalance(t *testing.T, helper *Helper, tx transaction.Transaction, diffs []*diffBalance) {
	sender, _ := tx.Sender()

	commissionPrice := tx.MulGasPrice(tx.Price(helper.app.CurrentState().Commission().GetCommissions()))
	initialCommissionCoinBalance := helper.app.CurrentState().Accounts().GetBalance(sender, tx.CommissionCoin())

	var initialBalances []*big.Int
	for _, diff := range diffs {
		initialBalances = append(initialBalances, helper.app.CurrentState().Accounts().GetBalance(sender, diff.Coin))
	}

	_, results := helper.NextBlock(tx)

	for _, resp := range results {
		if diffs != nil {
			if resp.Code != code.OK {
				t.Fatalf("Response code is not OK: %d, %s", resp.Code, resp.Log)
			}
		} else {
			if resp.Code == code.OK {
				t.Fatalf("Response code is OK, but wanted Error")
			}
			t.Run("failCommission", func(t *testing.T) {
				var cmp = 0
				var failPrice = big.NewInt(1)
				if tx.CommissionCoin() == types.USDTID {
					commissions := helper.app.CurrentState().Commission().GetCommissions()
					failPrice = tx.MulGasPrice(big.NewInt(0).Add(commissions.FailedTx, big.NewInt(0).Mul(big.NewInt(tx.PayloadAndServiceDataLen()), commissions.PayloadByte)))
				} else {
					cmp = 1
				}
				if big.NewInt(0).Sub(initialCommissionCoinBalance, helper.app.CurrentState().Accounts().GetBalance(sender, tx.CommissionCoin())).Cmp(failPrice) != cmp {
					t.Error("error sender usd balance")
				}
			})
			return
		}
	}

	var checkedCommissionCoin bool
	for i, diff := range diffs {
		t.Run(diff.Coin.String(), func(t *testing.T) {
			var cmp = 0
			newBalance := big.NewInt(0).Sub(initialBalances[i], diff.Value)
			if diff.Coin == tx.CommissionCoin() {
				checkedCommissionCoin = true
				if tx.CommissionCoin() != types.USDTID {
					commissionPrice = big.NewInt(1)
					cmp = 1
				}
				newBalance.Sub(newBalance, commissionPrice)
			}
			if newBalance.Cmp(helper.app.CurrentState().Accounts().GetBalance(sender, diff.Coin)) != cmp {
				t.Errorf("error sender CoinID-%d balance diff %s", diff.Coin, big.NewInt(0).Sub(newBalance, helper.app.CurrentState().Accounts().GetBalance(sender, diff.Coin)))
			}
		})

	}

	t.Run("commissionCoin", func(t *testing.T) {
		if !checkedCommissionCoin {
			var cmp = 0
			if tx.CommissionCoin() != types.USDTID {
				cmp = 1
			}
			if big.NewInt(0).Sub(initialCommissionCoinBalance, helper.app.CurrentState().Accounts().GetBalance(sender, tx.CommissionCoin())).Cmp(commissionPrice) != cmp {
				t.Error("error sender usd balance")
			}
		}
	})

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
