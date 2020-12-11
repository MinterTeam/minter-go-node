package transaction

import (
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestSellCoinTx(t *testing.T) {
	cState := getState()

	buyCoinID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	minValToBuy, _ := big.NewInt(0).SetString("957658277688702625", 10)
	data := SellCoinData{
		CoinToSell:        coin,
		ValueToSell:       helpers.BipToPip(big.NewInt(10)),
		CoinToBuy:         buyCoinID,
		MinimumValueToBuy: minValToBuy,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeSellCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)

	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999989900000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("999955002849793446", 10)
	testBalance := cState.Accounts.GetBalance(addr, buyCoinID)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", getTestCoinSymbol(), targetTestBalance, testBalance)
	}

	checkState(t, cState)
}

func TestSellCoinTxBaseToCustomBaseCommission(t *testing.T) {
	// sell_coin: MNT
	// buy_coin: TEST
	// gas_coin: MNT
	cState := getState()

	coinToSell := types.GetBaseCoinID()
	coinToBuy := types.StrToCoinSymbol("TEST")
	gasCoin := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toSell := helpers.BipToPip(big.NewInt(100))

	coinToBuyID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)

	tx := createSellCoinTx(coinToSell, coinToBuyID, gasCoin, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	estimatedBuyBalance := formula.CalculatePurchaseReturn(initialVolume, initialReserve, crr, toSell)
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins + commission
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, tx.CommissionInBaseCoin())
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToBuyID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Add(estimatedReserve, toSell)
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Add(estimatedSupply, formula.CalculatePurchaseReturn(initialVolume, initialReserve, crr, toSell))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	checkState(t, cState)
}

func TestSellCoinTxCustomToBaseBaseCommission(t *testing.T) {
	// sell_coin: TEST
	// buy_coin: MNT
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("TEST")
	coinToBuy := types.GetBaseCoinID()
	gasCoin := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toSell := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToSell)

	privateKey, addr := getAccount()
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume.Add(initialVolume, initialBalance)
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)

	tx := createSellCoinTx(coinToSellID, coinToBuy, gasCoin, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins + commission
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	estimatedBuyBalance := formula.CalculateSaleReturn(initialVolume, initialReserve, crr, toSell)
	estimatedBuyBalance.Add(estimatedBuyBalance, initialGasBalance)
	estimatedBuyBalance.Sub(estimatedBuyBalance, tx.CommissionInBaseCoin())
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToSellID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Sub(estimatedReserve, formula.CalculateSaleReturn(initialVolume, initialReserve, crr, toSell))
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Sub(estimatedSupply, toSell)
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	checkState(t, cState)
}

func TestSellCoinTxCustomToCustomBaseCommission(t *testing.T) {
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	gasCoin := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toSell := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	coinToBuyID, initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)
	initialVolume1.Add(initialVolume1, initialBalance)

	tx := createSellCoinTx(coinToSellID, coinToBuyID, gasCoin, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	estimatedBuyBalance := formula.CalculatePurchaseReturn(initialVolume2, initialReserve2, crr2, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSellID)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve")
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		estimatedSupply.Sub(estimatedSupply, toSell)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuyID)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve")
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, formula.CalculatePurchaseReturn(initialVolume2, initialReserve2, crr2, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell)))
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	checkState(t, cState)
}

func TestSellCoinTxBaseToCustomCustomCommission(t *testing.T) {
	// sell_coin: MNT
	// buy_coin: TEST
	// gas_coin: TEST

	coinToSell := types.GetBaseCoinID()
	coinToBuy := types.StrToCoinSymbol("TEST")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toSell := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToBuyID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Accounts.AddBalance(addr, coinToBuyID, initialGasBalance)
	cState.Coins.AddVolume(coinToBuyID, initialGasBalance)
	initialVolume.Add(initialVolume, initialGasBalance)

	tx := createSellCoinTx(coinToSell, coinToBuyID, coinToBuyID, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins + commission
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	estimatedReturn := formula.CalculatePurchaseReturn(initialVolume, initialReserve, crr, toSell)
	estimatedBuyBalance := big.NewInt(0).Set(estimatedReturn)
	estimatedBuyBalance.Add(estimatedBuyBalance, initialGasBalance)
	estimatedBuyBalance.Sub(estimatedBuyBalance, formula.CalculateSaleAmount(big.NewInt(0).Add(initialVolume, estimatedReturn), big.NewInt(0).Add(initialReserve, toSell), crr, tx.CommissionInBaseCoin()))
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToBuyID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Add(estimatedReserve, toSell)
	estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Add(estimatedSupply, formula.CalculatePurchaseReturn(initialVolume, initialReserve, crr, toSell))
	estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(big.NewInt(0).Add(initialVolume, estimatedReturn), big.NewInt(0).Add(initialReserve, toSell), crr, tx.CommissionInBaseCoin()))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	checkState(t, cState)
}

func TestSellCoinTxCustomToBaseCustomCommission(t *testing.T) {
	// sell_coin: TEST
	// buy_coin: MNT
	// gas_coin: TEST

	coinToSell := types.StrToCoinSymbol("TEST")
	coinToBuy := types.GetBaseCoinID()
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toSell := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToSell)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume.Add(initialVolume, initialBalance)

	tx := createSellCoinTx(coinToSellID, coinToBuy, coinToSellID, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	estimatedReturn := formula.CalculateSaleReturn(initialVolume, initialReserve, crr, toSell)
	estimatedBuyBalance := big.NewInt(0).Set(estimatedReturn)
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins + commission
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume, toSell), big.NewInt(0).Sub(initialReserve, estimatedReturn), crr, tx.CommissionInBaseCoin()))
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToSellID)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Sub(estimatedReserve, estimatedReturn)
	estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Sub(estimatedSupply, toSell)
	estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume, toSell), big.NewInt(0).Sub(initialReserve, estimatedReturn), crr, tx.CommissionInBaseCoin()))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}

	checkState(t, cState)
}

func TestSellCoinTxCustomToCustomCustom1Commission(t *testing.T) {
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: TEST1

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toSell := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	coinToBuyID, initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume1.Add(initialVolume1, initialBalance)

	tx := createSellCoinTx(coinToSellID, coinToBuyID, coinToSellID, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	bipReturn := formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell)
	estimatedBuyBalance := formula.CalculatePurchaseReturn(initialVolume2, initialReserve2, crr2, bipReturn)
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	commission := formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume1, toSell), big.NewInt(0).Sub(initialReserve1, bipReturn), crr1, tx.CommissionInBaseCoin())
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, commission)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSellID)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
		estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve")
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		estimatedSupply.Sub(estimatedSupply, toSell)
		estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuyID)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve")
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, formula.CalculatePurchaseReturn(initialVolume2, initialReserve2, crr2, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell)))
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	checkState(t, cState)
}

func TestSellCoinTxCustomToCustomCustom2Commission(t *testing.T) {
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: TEST2

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toSell := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	coinToSellID, initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	coinToBuyID, initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSellID, initialBalance)
	cState.Coins.AddVolume(coinToSellID, initialBalance)
	initialVolume1.Add(initialVolume1, initialBalance)

	cState.Accounts.AddBalance(addr, coinToBuyID, initialGasBalance)
	cState.Coins.AddVolume(coinToBuyID, initialGasBalance)
	initialVolume2.Add(initialVolume2, initialGasBalance)

	tx := createSellCoinTx(coinToSellID, coinToBuyID, coinToBuyID, toSell, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuyID)
	bipReturn := formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell)
	estimatedReturn := formula.CalculatePurchaseReturn(initialVolume2, initialReserve2, crr2, bipReturn)
	commission := formula.CalculateSaleAmount(big.NewInt(0).Add(initialVolume2, estimatedReturn), big.NewInt(0).Add(initialReserve2, bipReturn), crr2, tx.CommissionInBaseCoin())

	estimatedBuyBalance := big.NewInt(0).Set(estimatedReturn)
	estimatedBuyBalance.Sub(estimatedBuyBalance, commission)
	estimatedBuyBalance.Add(estimatedBuyBalance, initialGasBalance)
	if buyCoinBalance.Cmp(estimatedBuyBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyBalance.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSellID)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSellID)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve")
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		estimatedSupply.Sub(estimatedSupply, toSell)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuyID)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell))
		estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve")
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, formula.CalculatePurchaseReturn(initialVolume2, initialReserve2, crr2, formula.CalculateSaleReturn(initialVolume1, initialReserve1, crr1, toSell)))
		estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	checkState(t, cState)
}

func TestSellCoinTxToCoinSupplyOverflow(t *testing.T) {
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, sellCoinID := createTestCoin(cState), types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, sellCoinID, helpers.BipToPip(big.NewInt(100)))

	coinToBuy := cState.Coins.GetCoin(coinToBuyID)
	coinToBuy.CMaxSupply = big.NewInt(1)

	tx := createSellCoinTx(sellCoinID, coinToBuyID, sellCoinID, helpers.BipToPip(big.NewInt(90)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinSupplyOverflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinSupplyOverflow, response.Log)
	}

	checkState(t, cState)

	// custom buy and sell coins

	cState.Coins.Create(
		cState.App.GetNextCoinID(),
		types.StrToCoinSymbol("TEST9"),
		"TEST COIN",
		helpers.BipToPip(big.NewInt(100000)),
		10,
		helpers.BipToPip(big.NewInt(100000)),
		helpers.BipToPip(big.NewInt(1000000)),
		nil,
	)

	coinToSellID := cState.App.GetNextCoinID()
	cState.Accounts.AddBalance(types.Address{}, coinToSellID, helpers.BipToPip(big.NewInt(100000)))
	cState.App.SetCoinsCount(coinToSellID.Uint32())

	tx = createSellCoinTx(coinToSellID, coinToBuyID, coinToSellID, helpers.BipToPip(big.NewInt(90)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinSupplyOverflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinSupplyOverflow, response.Log)
	}

	checkState(t, cState)
}

func TestSellCoinTxToMinimumValueToBuyReached(t *testing.T) {
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, sellCoinID := createTestCoin(cState), types.GetBaseCoinID()

	valueToSell := big.NewInt(2e18)
	cState.Accounts.AddBalance(addr, sellCoinID, helpers.BipToPip(big.NewInt(2)))

	data := SellCoinData{
		CoinToBuy:         coinToBuyID,
		ValueToSell:       valueToSell,
		CoinToSell:        sellCoinID,
		MinimumValueToBuy: big.NewInt(9e18),
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx := &Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       sellCoinID,
		Type:          TypeSellCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
		decodedData:   data,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.MinimumValueToBuyReached {
		t.Fatalf("Response code is not %d. Error %s", code.MinimumValueToBuyReached, response.Log)
	}

	checkState(t, cState)

	// coin to buy == base coin

	cState.Accounts.SubBalance(types.Address{}, coinToBuyID, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coinToBuyID, helpers.BipToPip(big.NewInt(100000)))

	data.CoinToBuy = sellCoinID
	data.CoinToSell = coinToBuyID
	data.MinimumValueToBuy = big.NewInt(9e18)
	data.ValueToSell = big.NewInt(1)
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx.Data = encodedData
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.MinimumValueToBuyReached {
		t.Fatalf("Response code is not %d. Error %s", code.MinimumValueToBuyReached, response.Log)
	}

	checkState(t, cState)

	// custom buy and sell coins

	cState.Coins.Create(
		cState.App.GetNextCoinID(),
		types.StrToCoinSymbol("TEST9"),
		"TEST COIN",
		helpers.BipToPip(big.NewInt(100000)),
		10,
		helpers.BipToPip(big.NewInt(100000)),
		helpers.BipToPip(big.NewInt(1000000)),
		nil,
	)

	coinToSellID := cState.App.GetNextCoinID()
	cState.Accounts.AddBalance(types.Address{}, coinToSellID, helpers.BipToPip(big.NewInt(100000)))
	cState.App.SetCoinsCount(coinToSellID.Uint32())

	data.CoinToBuy = coinToBuyID
	data.CoinToSell = coinToSellID
	data.MinimumValueToBuy = big.NewInt(9e18)
	data.ValueToSell = big.NewInt(1)

	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx.Data = encodedData
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.MinimumValueToBuyReached {
		t.Fatalf("Response code is not %d. Error %s", code.MinimumValueToBuyReached, response.Log)
	}

	checkState(t, cState)
}

func TestSellCoinTxToCoinReserveNotSufficient(t *testing.T) {
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, coinToSellID := createTestCoin(cState), types.GetBaseCoinID()

	cState.Coins.Create(
		cState.App.GetNextCoinID(),
		types.StrToCoinSymbol("TEST9"),
		"TEST COIN",
		helpers.BipToPip(big.NewInt(100000)),
		10,
		helpers.BipToPip(big.NewInt(100000)),
		helpers.BipToPip(big.NewInt(1000000)),
		nil,
	)

	customCoinToSellID := cState.App.GetNextCoinID()
	cState.App.SetCoinsCount(customCoinToSellID.Uint32())

	cState.Accounts.AddBalance(types.Address{0}, customCoinToSellID, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coinToSellID, helpers.BipToPip(big.NewInt(5000000)))

	tx := createSellCoinTx(coinToBuyID, coinToSellID, coinToBuyID, helpers.BipToPip(big.NewInt(100000)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinReserveNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveNotSufficient, response.Log)
	}

	checkState(t, cState)

	// gas coin == coin to sell

	cState.Coins.SubReserve(coinToBuyID, helpers.BipToPip(big.NewInt(100000)))

	tx = createSellCoinTx(coinToBuyID, customCoinToSellID, coinToBuyID, helpers.BipToPip(big.NewInt(1)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinReserveNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveNotSufficient, response.Log)
	}

	checkState(t, cState)
}

func TestSellCoinTxInsufficientFunds(t *testing.T) {
	cState := getState()

	coinToBuyID, coinToSellID := createTestCoin(cState), types.GetBaseCoinID()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	tx := createSellCoinTx(coinToSellID, coinToBuyID, coinToSellID, helpers.BipToPip(big.NewInt(1000)), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	checkState(t, cState)
}

func TestSellCoinTxEqualCoins(t *testing.T) {
	cState := getState()
	coinID := createTestCoin(cState)
	privateKey, _ := crypto.GenerateKey()

	tx := createSellCoinTx(coinID, coinID, coinID, big.NewInt(1), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CrossConvert {
		t.Fatalf("Response code is not %d. Error %s", code.CrossConvert, response.Log)
	}

	checkState(t, cState)
}

func TestSellCoinTxToNonExistCoins(t *testing.T) {
	cState := getState()
	coinID := createTestCoin(cState)
	privateKey, _ := crypto.GenerateKey()

	tx := createSellCoinTx(5, coinID, coinID, big.NewInt(1), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	tx = createSellCoinTx(coinID, 5, coinID, big.NewInt(1), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	checkState(t, cState)

	tx = createSellCoinTx(coinID, types.GetBaseCoinID(), 5, big.NewInt(1), 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	checkState(t, cState)
}

func createSellCoinTx(sellCoin, buyCoin, gasCoin types.CoinID, valueToSell *big.Int, nonce uint64) *Transaction {
	data := SellCoinData{
		CoinToSell:        sellCoin,
		ValueToSell:       valueToSell,
		CoinToBuy:         buyCoin,
		MinimumValueToBuy: big.NewInt(0),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		panic(err)
	}

	return &Transaction{
		Nonce:         nonce,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       gasCoin,
		Type:          TypeSellCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,

		decodedData: data,
	}
}
