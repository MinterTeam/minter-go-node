package transaction

import (
	"math/big"
	"reflect"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestCreateMultisigTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	addresses := []types.Address{
		addr,
		addr2,
	}

	weights := []uint32{1, 1}

	data := CreateMultisigData{
		Threshold: 1,
		Weights:   weights,
		Addresses: addresses,
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
		Type:          TypeCreateMultisig,
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

	targetBalance, _ := big.NewInt(0).SetString("999999900000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	var multisigAddr types.Address

	for _, item := range response.Tags {
		if string(item.Key) == "tx.created_multisig" {
			multisigAddr = types.HexToAddress(string(item.Value))
		}
	}

	multisig := cState.Accounts.GetAccount(multisigAddr)

	if !multisig.IsMultisig() {
		t.Fatalf("Multisig %s is not created", multisigAddr.String())
	}

	msigData := multisig.Multisig()

	if !reflect.DeepEqual(msigData.Addresses, addresses) {
		t.Fatalf("Addresses are not correct")
	}

	if !reflect.DeepEqual(msigData.Weights, weights) {
		t.Fatalf("Weights are not correct")
	}

	if msigData.Threshold != 1 {
		t.Fatalf("Threshold is not correct")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateMultisigFromExistingAccountTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	addresses := []types.Address{
		addr,
		addr2,
	}

	weights := []uint32{1, 1}

	data := CreateMultisigData{
		Threshold: 1,
		Weights:   weights,
		Addresses: addresses,
	}

	msigAddr := accounts.CreateMultisigAddress(addr, 1)

	initialBalance := big.NewInt(10)
	cState.Accounts.AddBalance(msigAddr, types.GetBaseCoinID(), initialBalance)

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeCreateMultisig,
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

	targetBalance, _ := big.NewInt(0).SetString("999999900000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	var multisigAddr types.Address

	for _, item := range response.Tags {
		if string(item.Key) == "tx.created_multisig" {
			multisigAddr = types.HexToAddress(string(item.Value))
		}
	}

	multisig := cState.Accounts.GetAccount(multisigAddr)

	if !multisig.IsMultisig() {
		t.Fatalf("Multisig %s is not created", multisigAddr.String())
	}

	msigData := multisig.Multisig()

	if !reflect.DeepEqual(msigData.Addresses, addresses) {
		t.Fatalf("Addresses are not correct")
	}

	if !reflect.DeepEqual(msigData.Weights, weights) {
		t.Fatalf("Weights are not correct")
	}

	if msigData.Threshold != 1 {
		t.Fatalf("Threshold is not correct")
	}

	if cState.Accounts.GetBalance(msigAddr, types.GetBaseCoinID()).Cmp(initialBalance) != 0 {
		t.Fatalf("Msig balance was not persisted")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateExistingMultisigTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	data := CreateMultisigData{
		Threshold: 1,
		Weights:   []uint32{1, 1},
		Addresses: []types.Address{
			addr,
			addr2,
		},
	}

	cState.Accounts.CreateMultisig(data.Weights, data.Addresses, data.Threshold, accounts.CreateMultisigAddress(addr, 1))

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeCreateMultisig,
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

	if response.Code != code.MultisigExists {
		t.Fatalf("Response code is not %d. Got %d", code.MultisigExists, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateMultisigOwnersTxToNonExistAddress(t *testing.T) {
	cState := getState()

	addr := types.Address{0}
	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)

	data := EditMultisigData{
		Threshold: 3,
		Weights:   []uint32{2, 1, 2},
		Addresses: []types.Address{addr1, addr2, addr3},
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
		Type:          TypeEditMultisig,
		Data:          encodedData,
		SignatureType: SigTypeMulti,
	}

	tx.SetMultisigAddress(addr)

	if err := tx.Sign(privateKey3); err != nil {
		t.Fatal(err)
	}

	response := data.basicCheck(&tx, state.NewCheckState(cState))
	if response.Code != code.MultisigNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.MultisigNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateMultisigOwnersTxToTooLargeOwnersList(t *testing.T) {
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)

	coin := types.GetBaseCoinID()

	weights := make([]uint32, 33)
	for i := uint32(0); i <= 32; i++ {
		weights[i] = i
	}

	data := CreateMultisigData{
		Threshold: 3,
		Weights:   weights,
		Addresses: []types.Address{addr1, addr2, addr3},
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
		Type:          TypeCreateMultisig,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey3); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.TooLargeOwnersList {
		t.Fatalf("Response code is not %d. Error %s", code.TooLargeOwnersList, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateMultisigOwnersTxIncorrectWeights(t *testing.T) {
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)

	data := CreateMultisigData{
		Threshold: 3,
		Weights:   []uint32{1, 2, 3, 4},
		Addresses: []types.Address{addr1, addr2, addr3},
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
		Type:          TypeCreateMultisig,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey3); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.DifferentCountAddressesAndWeights {
		t.Fatalf("Response code is not %d. Error %s", code.DifferentCountAddressesAndWeights, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	data.Weights = []uint32{1, 2, 1024}
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx.Data = encodedData
	if err := tx.Sign(privateKey3); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.IncorrectWeights {
		t.Fatalf("Response code is not %d. Error %s", code.IncorrectWeights, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateMultisigOwnersTxToAddressDuplication(t *testing.T) {
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)

	data := CreateMultisigData{
		Threshold: 3,
		Weights:   []uint32{1, 2, 3},
		Addresses: []types.Address{addr1, addr1, addr3},
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
		Type:          TypeCreateMultisig,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey3); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.DuplicatedAddresses {
		t.Fatalf("Response code is not %d. Error %s", code.DuplicatedAddresses, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateMultisigOwnersTxToInsufficientFunds(t *testing.T) {
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)

	data := CreateMultisigData{
		Threshold: 3,
		Weights:   []uint32{1, 2, 3},
		Addresses: []types.Address{addr1, addr2, addr3},
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
		Type:          TypeCreateMultisig,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey3); err != nil {
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

func TestCreateMultisigTxToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)

	customCoin := createTestCoin(cState)
	cState.Coins.SubReserve(customCoin, helpers.BipToPip(big.NewInt(90000)))

	cState.Accounts.AddBalance(addr3, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(1000000)))

	data := CreateMultisigData{
		Threshold: 3,
		Weights:   []uint32{1, 2, 3},
		Addresses: []types.Address{addr1, addr2, addr3},
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       customCoin,
		Type:          TypeCreateMultisig,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey3); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s, info %s", code.CommissionCoinNotSufficient, response.Log, response.Info)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
