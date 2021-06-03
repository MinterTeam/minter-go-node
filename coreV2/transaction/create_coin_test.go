package transaction

import (
	"encoding/binary"
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestCreateCoinTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	_, _, err = cState.Tree().Commit(cState.Coins)
	if err != nil {
		t.Fatalf("Commit coins failed. Error %s", err)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	targetBalance, _ := big.NewInt(0).SetString("989000000000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Errorf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	stateCoin := cState.Coins.GetCoinBySymbol(toCreate, 0)

	if stateCoin == nil {
		t.Fatalf("Coin %s not found in state", toCreate)
	}

	if stateCoin.Reserve().Cmp(reserve) != 0 {
		t.Fatalf("Reserve balance in state is not correct. Expected %s, got %s", reserve, stateCoin.Reserve())
	}

	if stateCoin.Volume().Cmp(amount) != 0 {
		t.Fatalf("Volume in state is not correct. Expected %s, got %s", amount, stateCoin.Volume())
	}

	if stateCoin.Crr() != crr {
		t.Fatalf("Crr in state is not correct. Expected %d, got %d", crr, stateCoin.Crr())
	}

	if stateCoin.Name() != name {
		t.Fatalf("Name in state is not correct. Expected %s, got %s", name, stateCoin.Name())
	}

	if stateCoin.Version() != 0 {
		t.Fatalf("Version in state is not correct. Expected %d, got %d", 0, stateCoin.Version())
	}

	symbolInfo := cState.Coins.GetSymbolInfo(toCreate)
	if symbolInfo == nil {
		t.Fatalf("Symbol %s info not found in state", toCreate)
	}

	if *symbolInfo.OwnerAddress() != addr {
		t.Fatalf("Target owner address is not correct. Expected %s, got %s", addr.String(), symbolInfo.OwnerAddress().String())
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinWithIncorrectName(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)

	var name [65]byte
	binary.BigEndian.PutUint64(name[:], 0)

	data := CreateCoinData{
		Name:                 string(name[:]),
		Symbol:               toCreate,
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.InvalidCoinName {
		t.Fatalf("Response code is not %d. Error %s", code.InvalidCoinName, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinWithInvalidSymbol(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("ABC-DEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.InvalidCoinSymbol {
		t.Fatalf("Response code is not %d. Error %s", code.InvalidCoinSymbol, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinWithExistingSymbol(t *testing.T) {
	t.Parallel()
	cState := getState()

	createTestCoin(cState)
	cState.Commit()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("TEST")
	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CoinAlreadyExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinAlreadyExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinWithWrongCrr(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(9)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongCrr {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCrr, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	data.ConstantReserveRatio = uint32(101)

	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx = Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeCreateCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongCrr {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCrr, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinWithWrongCoinSupply(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	crr := uint32(50)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        big.NewInt(1),
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            helpers.BipToPip(big.NewInt(100000)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	data.InitialAmount = helpers.BipToPip(big.NewInt(1000000))
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx = Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeCreateCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	data.MaxSupply = big.NewInt(0).Exp(big.NewInt(100), big.NewInt(15+18), nil)
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

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
	}

	data.MaxSupply = maxCoinSupply
	data.InitialReserve = helpers.BipToPip(big.NewInt(1000))
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

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinGas(t *testing.T) {
	t.Parallel()
	// t.SkipNow()
	data := CreateCoinData{
		Symbol: types.StrToCoinSymbol("ABC"),
	}

	if data.CommissionData(&commissionPrice).String() != "1000000000000000000000000" {
		t.Fatal("CommissionData for symbol with length 3 is not correct.")
	}

	data.Symbol = types.StrToCoinSymbol("ABCD")
	if data.CommissionData(&commissionPrice).String() != "100000000000000000000000" {
		t.Fatal("CommissionData for symbol with length 4 is not correct.")
	}

	data.Symbol = types.StrToCoinSymbol("ABCDE")
	if data.CommissionData(&commissionPrice).String() != "10000000000000000000000" {
		t.Fatal("CommissionData for symbol with length 5 is not correct.")
	}

	data.Symbol = types.StrToCoinSymbol("ABCDEF")
	if data.CommissionData(&commissionPrice).String() != "1000000000000000000000" {
		t.Fatal("CommissionData for symbol with length 6 is not correct.")
	}

	data.Symbol = types.StrToCoinSymbol("ABCDEFG")
	if data.CommissionData(&commissionPrice).String() != "100000000000000000000" {
		t.Fatal("CommissionData for symbol with length 7 is not correct.")
	}
}

func TestCreateCoinWithInsufficientFundsForGas(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	crr := uint32(50)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        helpers.BipToPip(big.NewInt(10)),
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinTxToGasCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	customCoin := createTestCoin(cState)
	cState.Coins.SubReserve(customCoin, helpers.BipToPip(big.NewInt(90000)))

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	crr := uint32(50)
	name := "My Test Coin"

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Coins.AddVolume(customCoin, helpers.BipToPip(big.NewInt(1050)))
	cState.Accounts.AddBalance(addr, customCoin, helpers.BipToPip(big.NewInt(1050)))
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(10000)))
	cState.Commit()

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        helpers.BipToPip(big.NewInt(10)),
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinToInsufficientFundsForGasCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	crr := uint32(50)
	name := "My Test Coin"

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(10000)))

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        helpers.BipToPip(big.NewInt(10)),
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %d %s", code.InsufficientFunds, response.Code, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinToInsufficientFundsForInitialReserve(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(11000))
	crr := uint32(50)
	name := "My Test Coin"

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(10000)))

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        helpers.BipToPip(big.NewInt(10)),
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCreateCoinToSameSymbolInOneBlock(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr2, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("TEST")
	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)
	name := "My Test Coin"

	data := CreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
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
		Type:          TypeCreateCoin,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != 0 {
		t.Fatalf("Response code is not success. Error %s", response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	if err := tx.Sign(privateKey2); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CoinAlreadyExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinAlreadyExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
