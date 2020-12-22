package transaction

import (
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestSellAllCoinTx(t *testing.T) {
	cState := getState()

	coinID := createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	minValToBuy, _ := big.NewInt(0).SetString("151191152412701306252", 10)
	data := SellAllCoinData{
		CoinToSell:        coin,
		CoinToBuy:         coinID,
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
		Type:          TypeSellAllCoin,
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
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(types.Big0) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, types.Big0, balance)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("27098160365576186275223", 10)
	testBalance := cState.Accounts.GetBalance(addr, coinID)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", getTestCoinSymbol(), targetTestBalance, testBalance)
	}

	checkState(t, cState)
}

func TestSellAllCoinTxWithSameCoins(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	minValToBuy, _ := big.NewInt(0).SetString("151191152412701306252", 10)
	data := SellAllCoinData{
		CoinToSell:        coin,
		CoinToBuy:         coin,
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
		Type:          TypeSellAllCoin,
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
	if response.Code != code.CrossConvert {
		t.Fatalf("Response code is not %d. Error %s", code.CrossConvert, response.Log)
	}

	checkState(t, cState)
}

func TestSellAllCoinTxWithInvalidCoins(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	coin := types.CoinID(5)

	minValToBuy, _ := big.NewInt(0).SetString("151191152412701306252", 10)
	data := SellAllCoinData{
		CoinToSell:        coin,
		CoinToBuy:         types.GetBaseCoinID(),
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
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSellAllCoin,
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
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	checkState(t, cState)

	data.CoinToSell = types.GetBaseCoinID()
	data.CoinToBuy = types.CoinID(5)
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
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
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	checkState(t, cState)
}

func TestSellAllCoinTxWithMinimumValueToBuy(t *testing.T) {
	cState := getState()
	coinID := createTestCoin(cState)
	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()

	minValToBuy, _ := big.NewInt(0).SetString("151191152412701306252", 10)
	data := SellAllCoinData{
		CoinToSell:        coin,
		CoinToBuy:         coinID,
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
		Type:          TypeSellAllCoin,
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
	if response.Code != code.MinimumValueToBuyReached {
		t.Fatalf("Response code is not %d. Error %s", code.MinimumValueToBuyReached, response.Log)
	}

	checkState(t, cState)
}

func TestSellAllCoinTxWithInsufficientFunds(t *testing.T) {
	cState := getState()
	coinID := createTestCoin(cState)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	minValToBuy, _ := big.NewInt(0).SetString("0", 10)
	data := SellAllCoinData{
		CoinToSell:        coinID,
		CoinToBuy:         coin,
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
		Type:          TypeSellAllCoin,
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
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	checkState(t, cState)

	nextCoinID := cState.App.GetNextCoinID()
	cState.Coins.Create(
		nextCoinID,
		types.StrToCoinSymbol("TEST9"),
		"TEST COIN",
		helpers.BipToPip(big.NewInt(100000)),
		10,
		helpers.BipToPip(big.NewInt(100000)),
		helpers.BipToPip(big.NewInt(1000000)),
		nil,
	)

	coinToSellID := nextCoinID
	cState.App.SetCoinsCount(nextCoinID.Uint32())
	cState.Accounts.AddBalance(types.Address{}, coinToSellID, big.NewInt(0).Sub(helpers.BipToPip(big.NewInt(100000)), big.NewInt(1)))
	cState.Accounts.AddBalance(addr, coinToSellID, big.NewInt(1))

	data.CoinToBuy = coinID
	data.CoinToSell = coinToSellID
	data.MinimumValueToBuy = big.NewInt(9e18)

	tx.Data, err = rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	checkState(t, cState)
}

func TestSellAllCoinTxToCoinSupplyOverflow(t *testing.T) {
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, sellCoinID := createTestCoin(cState), types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, sellCoinID, helpers.BipToPip(big.NewInt(100)))

	coinToBuy := cState.Coins.GetCoin(coinToBuyID)
	coinToBuy.CMaxSupply = big.NewInt(1)

	data := SellAllCoinData{
		CoinToSell:        sellCoinID,
		CoinToBuy:         coinToBuyID,
		MinimumValueToBuy: big.NewInt(0),
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       data.CoinToSell,
		Type:          TypeSellAllCoin,
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
	cState.App.SetCoinsCount(coinToSellID.Uint32())
	cState.Accounts.AddBalance(addr, coinToSellID, helpers.BipToPip(big.NewInt(100000)))

	data.CoinToBuy = coinToBuyID
	data.CoinToSell = coinToSellID
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx.GasCoin = coinToSellID
	tx.Data = encodedData
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

func TestSellAllCoinTxToMinimumValueToBuyReached(t *testing.T) {
	cState := getState()
	privateKey, addr := getAccount()
	coinToBuyID, sellCoinID := createTestCoin(cState), types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, sellCoinID, helpers.BipToPip(big.NewInt(2)))

	data := SellAllCoinData{
		CoinToBuy:         coinToBuyID,
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
		Type:          TypeSellAllCoin,
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

	cState.Accounts.SubBalance(types.Address{}, coinToBuyID, big.NewInt(1))
	cState.Accounts.AddBalance(addr, coinToBuyID, big.NewInt(1))

	data.CoinToBuy = sellCoinID
	data.CoinToSell = coinToBuyID
	data.MinimumValueToBuy = big.NewInt(9e18)
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
	cState.App.SetCoinsCount(coinToSellID.Uint32())

	cState.Accounts.AddBalance(types.Address{}, coinToSellID, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coinToSellID, big.NewInt(1e17))
	cState.Accounts.AddBalance(addr, coinToSellID, big.NewInt(1e17))

	data.CoinToBuy = coinToBuyID
	data.CoinToSell = coinToSellID
	data.MinimumValueToBuy = big.NewInt(9e18)

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
