package transaction

import (
	"bytes"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tm-db"
	"github.com/xujiajun/nutsdb"
	"math/big"
	"os"
	"sync"
	"testing"
)

var (
	cdc = amino.NewCodec()
)

func init() {
	log.InitLog(config.GetConfig())
}

func getState() *state.State {
	opt := nutsdb.DefaultOptions
	opt.Dir = "/tmp/nutsdb"
	os.RemoveAll(opt.Dir)
	nuts, err := nutsdb.Open(opt)

	s, err := state.NewState(0, db.NewMemDB(), nuts, nil)

	if err != nil {
		panic(err)
	}

	return s
}

func getTestCoinSymbol() types.CoinSymbol {
	var coin types.CoinSymbol
	copy(coin[:], []byte("TEST"))

	return coin
}

func createTestCoin(stateDB *state.State) {
	volume := helpers.BipToPip(big.NewInt(100))
	reserve := helpers.BipToPip(big.NewInt(100))

	stateDB.Coins.Create(getTestCoinSymbol(), "TEST COIN", volume, 10, reserve)
}

func TestBuyCoinTx(t *testing.T) {
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999840525753990000000000", 10)
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}
}

func TestBuyCoinTxJSON(t *testing.T) {
	out := []byte("{\"coin_to_buy\":\"BIP\",\"value_to_buy\":\"1\",\"coin_to_sell\":\"TEST\",\"maximum_value_to_sell\":\"1\"}")

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
