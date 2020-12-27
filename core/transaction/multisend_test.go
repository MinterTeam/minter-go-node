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

func TestMultisendTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := MultisendData{
		List: []MultisendDataItem{
			{
				Coin:  coin,
				To:    to,
				Value: value,
			},
		},
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       coin,
		ChainID:       types.CurrentChainID,
		Type:          TypeMultisend,
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

	targetBalance, _ := big.NewInt(0).SetString("999989990000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetBalance, balance)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("10000000000000000000", 10)
	testBalance := cState.Accounts.GetBalance(to, coin)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", to.String(), targetTestBalance, testBalance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultisendTxToInvalidDataLength(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()

	data := MultisendData{
		List: nil,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       coin,
		ChainID:       types.CurrentChainID,
		Type:          TypeMultisend,
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
	if response.Code != code.InvalidMultisendData {
		t.Fatalf("Response code is not %d. Error %s", code.InvalidMultisendData, response.Log)
	}

	list := make([]MultisendDataItem, 101)
	for i := 0; i <= 100; i++ {
		list[i] = MultisendDataItem{
			Coin:  types.GetBaseCoinID(),
			To:    [20]byte{1},
			Value: helpers.BipToPip(big.NewInt(10)),
		}
	}

	data.List = list
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
	if response.Code != code.InvalidMultisendData {
		t.Fatalf("Response code is not %d. Error %s", code.InvalidMultisendData, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultisendTxToInsufficientFunds(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := MultisendData{
		List: []MultisendDataItem{
			{
				Coin:  coin,
				To:    to,
				Value: value,
			},
		},
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       coin,
		ChainID:       types.CurrentChainID,
		Type:          TypeMultisend,
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

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultisendToInvalidCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.CoinID(5)

	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := MultisendData{
		List: []MultisendDataItem{
			{
				Coin:  coin,
				To:    to,
				Value: value,
			},
		},
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		ChainID:       types.CurrentChainID,
		Type:          TypeMultisend,
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

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultisendToInsufficientReserve(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.CoinID(5)

	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := MultisendData{
		List: []MultisendDataItem{
			{
				Coin:  coin,
				To:    to,
				Value: value,
			},
		},
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		ChainID:       types.CurrentChainID,
		Type:          TypeMultisend,
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

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultisendTxToGasCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)

	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Commit()

	value := helpers.BipToPip(big.NewInt(10))
	to := types.Address([20]byte{1})

	data := MultisendData{
		List: []MultisendDataItem{
			{
				Coin:  coin,
				To:    to,
				Value: value,
			},
		},
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       coin,
		ChainID:       types.CurrentChainID,
		Type:          TypeMultisend,
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
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
