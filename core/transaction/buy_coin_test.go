package transaction

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tm-db"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	cdc = amino.NewCodec()

	rnd = rand.New(rand.NewSource(time.Now().Unix()))
)

func getState() *state.State {
	s, err := state.NewState(0, db.NewMemDB(), nil, 1, 1)

	if err != nil {
		panic(err)
	}

	return s
}

func getTestCoinSymbol() types.CoinSymbol {
	var coin types.CoinSymbol
	copy(coin[:], "TEST")

	return coin
}

func createTestCoin(stateDB *state.State) {
	volume := helpers.BipToPip(big.NewInt(100000))
	reserve := helpers.BipToPip(big.NewInt(100000))

	stateDB.Coins.Create(getTestCoinSymbol(), "TEST COIN", volume, 10, reserve, big.NewInt(0).Mul(volume, big.NewInt(10)))
}

func TestBuyCoinTxBaseToCustom(t *testing.T) {
	cState := getState()

	createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toBuy := helpers.BipToPip(big.NewInt(10))
	maxValToSell, _ := big.NewInt(0).SetString("159374246010000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          getTestCoinSymbol(),
		ValueToBuy:         toBuy,
		CoinToSell:         coin,
		MaximumValueToSell: maxValToSell,
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
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999899854987997899747979", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	testBalance := cState.Accounts.GetBalance(addr, getTestCoinSymbol())
	if testBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", getTestCoinSymbol(), toBuy, testBalance)
	}
}

func TestBuyCoinTxInsufficientFunds(t *testing.T) {
	cState := getState()

	createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	toBuy := helpers.BipToPip(big.NewInt(10))
	maxValToSell, _ := big.NewInt(0).SetString("159374246010000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          getTestCoinSymbol(),
		ValueToBuy:         toBuy,
		CoinToSell:         coin,
		MaximumValueToSell: maxValToSell,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:    1,
		GasPrice: 1,

		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}
}

func TestBuyCoinTxEqualCoins(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  getTestCoinSymbol(),
		ValueToBuy: big.NewInt(0),
		CoinToSell: getTestCoinSymbol(),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoin(),
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != code.CrossConvert {
		t.Fatalf("Response code is not %d. Error %s", code.CrossConvert, response.Log)
	}
}

func TestBuyCoinTxNotExistsBuyCoin(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  types.CoinSymbol{},
		ValueToBuy: big.NewInt(0),
		CoinToSell: types.GetBaseCoin(),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoin(),
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}
}

func TestBuyCoinTxNotExistsSellCoin(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  types.GetBaseCoin(),
		ValueToBuy: big.NewInt(0),
		CoinToSell: types.CoinSymbol{},
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoin(),
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}
}

func TestBuyCoinTxNotExistsGasCoin(t *testing.T) {
	cState := getState()

	createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()

	data := BuyCoinData{
		CoinToBuy:  types.GetBaseCoin(),
		ValueToBuy: big.NewInt(0),
		CoinToSell: getTestCoinSymbol(),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.CoinSymbol{},
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}
}

func TestBuyCoinTxNotGasCoin(t *testing.T) {
	cState := getState()

	createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, getTestCoinSymbol(), helpers.BipToPip(big.NewInt(1000)))

	data := BuyCoinData{
		CoinToBuy:          types.GetBaseCoin(),
		ValueToBuy:         big.NewInt(1),
		CoinToSell:         getTestCoinSymbol(),
		MaximumValueToSell: big.NewInt(10004502852067863),
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       getTestCoinSymbol(),
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}
}

func TestBuyCoinTxJSON(t *testing.T) {
	str := "{\"coin_to_buy\":\"%s\",\"value_to_buy\":\"1\",\"coin_to_sell\":\"TEST\",\"maximum_value_to_sell\":\"1\"}"
	out := []byte(fmt.Sprintf(str, types.GetBaseCoin().String()))

	buyCoinData := BuyCoinData{
		CoinToBuy:          types.GetBaseCoin(),
		ValueToBuy:         big.NewInt(1),
		CoinToSell:         getTestCoinSymbol(),
		MaximumValueToSell: big.NewInt(1),
	}

	result, err := cdc.MarshalJSON(buyCoinData)

	if err != nil {
		t.Fatalf("Error: %s", err.Error())
	}

	if !bytes.Equal(out, result) {
		t.Fatalf("Error: result is not correct %s, expected %s", string(result), string(out))
	}
}

func TestBuyCoinTxCustomToBase(t *testing.T) {
	cState := getState()

	createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := getTestCoinSymbol()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(10000000)))

	toBuy := helpers.BipToPip(big.NewInt(10))
	maxValToSell, _ := big.NewInt(0).SetString("159374246010000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          types.GetBaseCoin(),
		ValueToBuy:         toBuy,
		CoinToSell:         coin,
		MaximumValueToSell: maxValToSell,
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
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("9999998989954092563427063", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	baseBalance := cState.Accounts.GetBalance(addr, types.GetBaseCoin())
	if baseBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", types.GetBaseCoin(), toBuy, baseBalance)
	}

	coinData := cState.Coins.GetCoin(coin)

	targetReserve, _ := big.NewInt(0).SetString("99989900000000000000000", 10)
	if coinData.Reserve().Cmp(targetReserve) != 0 {
		t.Fatalf("Target %s reserve is not correct. Expected %s, got %s", coin, targetBalance, coinData.Reserve())
	}

	targetVolume, _ := big.NewInt(0).SetString("99998989954092563427063", 10)
	if coinData.Volume().Cmp(targetVolume) != 0 {
		t.Fatalf("Target %s volume is not correct. Expected %s, got %s", coin, targetVolume, coinData.Volume())
	}
}

func TestBuyCoinReserveUnderflow(t *testing.T) {
	cState := getState()

	createTestCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := getTestCoinSymbol()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(10000000)))

	toBuy := helpers.BipToPip(big.NewInt(99000))
	maxValToSell, _ := big.NewInt(0).SetString("36904896537720035723223", 10)
	data := BuyCoinData{
		CoinToBuy:          types.GetBaseCoin(),
		ValueToBuy:         toBuy,
		CoinToSell:         coin,
		MaximumValueToSell: maxValToSell,
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
		Type:          TypeBuyCoin,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != code.CoinReserveUnderflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveUnderflow, response.Log)
	}
}

func TestBuyCoinTxBaseToCustomBaseCommission(t *testing.T) {
	// sell_coin: MNT
	// buy_coin: TEST
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("MNT")
	coinToBuy := types.StrToCoinSymbol("TEST")
	gasCoin := types.StrToCoinSymbol("MNT")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins + commission
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, tx.CommissionInBaseCoin())
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy))
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToBuy)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy))
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Add(estimatedSupply, toBuy)
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}
}

func TestBuyCoinTxCustomToBaseBaseCommission(t *testing.T) {
	// sell_coin: TEST
	// buy_coin: MNT
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("TEST")
	coinToBuy := types.StrToCoinSymbol("MNT")
	gasCoin := types.StrToCoinSymbol("MNT")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance, _ := big.NewInt(0).SetString("100000000000000000", 10)
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToSell)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, formula.CalculateSaleAmount(initialVolume, initialReserve, crr, toBuy))
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToSell)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Sub(estimatedReserve, toBuy)
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume, initialReserve, crr, toBuy))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}
}

func TestBuyCoinTxCustomToCustomBaseCommission(t *testing.T) {
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: MNT

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST12")
	gasCoin := types.StrToCoinSymbol("MNT")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance, _ := big.NewInt(0).SetString("100000000000000000", 10)

	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	toSellBaseCoin := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
	toSell := formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, toSellBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToSell)

	estimatedReserve := big.NewInt(0).Set(initialReserve1)
	estimatedReserve.Sub(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume1)
	estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)))
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply")
	}
}

func TestBuyCoinTxBaseToCustomCustomCommission(t *testing.T) {
	// sell_coin: MNT
	// buy_coin: TEST
	// gas_coin: TEST

	coinToSell := types.StrToCoinSymbol("MNT")
	coinToBuy := types.StrToCoinSymbol("TEST")
	gasCoin := types.StrToCoinSymbol("TEST")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins + commission
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	estimatedBuyCoinBalance := big.NewInt(0).Set(toBuy)
	estimatedBuyCoinBalance.Add(estimatedBuyCoinBalance, initialGasBalance)
	toReserve := formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy)
	commission := formula.CalculateSaleAmount(big.NewInt(0).Add(initialVolume, toBuy), big.NewInt(0).Add(initialReserve, toReserve), crr, tx.CommissionInBaseCoin())
	estimatedBuyCoinBalance.Sub(estimatedBuyCoinBalance, commission)
	if buyCoinBalance.Cmp(estimatedBuyCoinBalance) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", estimatedBuyCoinBalance.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy))
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct")
	}

	// check reserve and supply
	coinData := cState.Coins.GetCoin(coinToBuy)

	estimatedReserve := big.NewInt(0).Set(initialReserve)
	estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume, initialReserve, crr, toBuy))
	estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
	if coinData.Reserve().Cmp(estimatedReserve) != 0 {
		t.Fatalf("Wrong coin reserve")
	}

	estimatedSupply := big.NewInt(0).Set(initialVolume)
	estimatedSupply.Add(estimatedSupply, toBuy)
	estimatedSupply.Sub(estimatedSupply, commission)
	if coinData.Volume().Cmp(estimatedSupply) != 0 {
		t.Fatalf("Wrong coin supply. Expected %s, got %s", estimatedSupply.String(), coinData.Volume().String())
	}
}

func TestBuyCoinTxCustomToBaseCustomCommission(t *testing.T) {
	// sell_coin: TEST
	// buy_coin: MNT
	// gas_coin: TEST

	coinToSell := types.StrToCoinSymbol("TEST")
	coinToBuy := types.StrToCoinSymbol("MNT")
	gasCoin := types.StrToCoinSymbol("TEST")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume, initialReserve, crr := createTestCoinWithSymbol(cState, coinToSell)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	shouldGive := formula.CalculateSaleAmount(initialVolume, initialReserve, crr, big.NewInt(0).Add(toBuy, tx.CommissionInBaseCoin()))
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, shouldGive)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSell)

		estimatedReserve := big.NewInt(0).Set(initialReserve)
		estimatedReserve.Sub(estimatedReserve, toBuy)
		estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume)
		estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume, initialReserve, crr, big.NewInt(0).Add(toBuy, tx.CommissionInBaseCoin())))
		//estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply. Expected %s, got %s", estimatedSupply.String(), coinData.Volume().String())
		}
	}
}

func TestBuyCoinTxCustomToCustomCustom1Commission(t *testing.T) {
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: TEST1

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	gasCoin := types.StrToCoinSymbol("TEST1")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct")
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	toSellBaseCoin := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
	toSell := formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, toSellBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	commission := formula.CalculateSaleAmount(big.NewInt(0).Sub(initialVolume1, toSell), big.NewInt(0).Sub(initialReserve1, toSellBaseCoin), crr1, tx.CommissionInBaseCoin())
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, commission)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSell)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
		estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)))
		estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuy)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, toBuy)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}
}

func TestBuyCoinTxCustomToCustomCustom2Commission(t *testing.T) {
	// sell_coin: TEST1
	// buy_coin: TEST2
	// gas_coin: TEST2

	coinToSell := types.StrToCoinSymbol("TEST1")
	coinToBuy := types.StrToCoinSymbol("TEST2")
	gasCoin := types.StrToCoinSymbol("TEST2")
	initialBalance := helpers.BipToPip(big.NewInt(10000000))
	initialGasBalance := helpers.BipToPip(big.NewInt(1))
	toBuy := helpers.BipToPip(big.NewInt(100))

	cState := getState()
	initialVolume1, initialReserve1, crr1 := createTestCoinWithSymbol(cState, coinToSell)
	initialVolume2, initialReserve2, crr2 := createTestCoinWithSymbol(cState, coinToBuy)

	privateKey, addr := getAccount()
	cState.Accounts.AddBalance(addr, coinToSell, initialBalance)
	cState.Accounts.AddBalance(addr, gasCoin, initialGasBalance)

	tx := createBuyCoinTx(coinToSell, coinToBuy, gasCoin, toBuy, 1)
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	// check response
	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.OK {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	// check received coins
	buyCoinBalance := cState.Accounts.GetBalance(addr, coinToBuy)
	buyCoinBalance.Sub(buyCoinBalance, initialGasBalance)
	toReserve := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
	commission := formula.CalculateSaleAmount(big.NewInt(0).Add(initialVolume2, toBuy), big.NewInt(0).Add(initialReserve2, toReserve), crr2, tx.CommissionInBaseCoin())
	buyCoinBalance.Add(buyCoinBalance, commission)
	if buyCoinBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Buy coin balance is not correct. Expected %s, got %s", toBuy.String(), buyCoinBalance.String())
	}

	// check sold coins
	sellCoinBalance := cState.Accounts.GetBalance(addr, coinToSell)
	estimatedSellCoinBalance := big.NewInt(0).Set(initialBalance)
	toSellBaseCoin := formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)
	toSell := formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, toSellBaseCoin)
	estimatedSellCoinBalance.Sub(estimatedSellCoinBalance, toSell)
	if sellCoinBalance.Cmp(estimatedSellCoinBalance) != 0 {
		t.Fatalf("Sell coin balance is not correct. Expected %s, got %s", estimatedSellCoinBalance.String(), sellCoinBalance.String())
	}

	// check reserve and supply
	{
		coinData := cState.Coins.GetCoin(coinToSell)

		estimatedReserve := big.NewInt(0).Set(initialReserve1)
		estimatedReserve.Sub(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume1)
		estimatedSupply.Sub(estimatedSupply, formula.CalculateSaleAmount(initialVolume1, initialReserve1, crr1, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy)))
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}

	{
		coinData := cState.Coins.GetCoin(coinToBuy)

		estimatedReserve := big.NewInt(0).Set(initialReserve2)
		estimatedReserve.Add(estimatedReserve, formula.CalculatePurchaseAmount(initialVolume2, initialReserve2, crr2, toBuy))
		estimatedReserve.Sub(estimatedReserve, tx.CommissionInBaseCoin())
		if coinData.Reserve().Cmp(estimatedReserve) != 0 {
			t.Fatalf("Wrong coin reserve. Expected %s, got %s", estimatedReserve.String(), coinData.Reserve().String())
		}

		estimatedSupply := big.NewInt(0).Set(initialVolume2)
		estimatedSupply.Add(estimatedSupply, toBuy)
		estimatedSupply.Sub(estimatedSupply, commission)
		if coinData.Volume().Cmp(estimatedSupply) != 0 {
			t.Fatalf("Wrong coin supply")
		}
	}
}

func createBuyCoinTx(sellCoin, buyCoin, gasCoin types.CoinSymbol, valueToBuy *big.Int, nonce uint64) *Transaction {
	maxValToSell, _ := big.NewInt(0).SetString("100000000000000000000000000000", 10)
	data := BuyCoinData{
		CoinToBuy:          buyCoin,
		ValueToBuy:         valueToBuy,
		CoinToSell:         sellCoin,
		MaximumValueToSell: maxValToSell,
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
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,

		decodedData: data,
	}
}

func getAccount() (*ecdsa.PrivateKey, types.Address) {
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	return privateKey, addr
}

func createTestCoinWithSymbol(stateDB *state.State, symbol types.CoinSymbol) (*big.Int, *big.Int, uint) {
	volume := helpers.BipToPip(big.NewInt(100000))
	reserve := helpers.BipToPip(big.NewInt(100000))
	volume.Mul(volume, big.NewInt(int64(rnd.Intn(9))+1))
	reserve.Mul(reserve, big.NewInt(int64(rnd.Intn(9))+1))

	crr := uint(10 + rnd.Intn(90))

	stateDB.Coins.Create(symbol, "TEST COIN", volume, crr, reserve, big.NewInt(0).Mul(volume, big.NewInt(10)))

	return big.NewInt(0).Set(volume), big.NewInt(0).Set(reserve), crr
}
