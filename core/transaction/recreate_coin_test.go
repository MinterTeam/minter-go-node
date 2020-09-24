package transaction

import (
	"crypto/ecdsa"
	"encoding/binary"
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestRecreateCoinTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := createTestCoinWithOwner(cState, addr) + 1
	gasCoin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, gasCoin, helpers.BipToPip(big.NewInt(20000)))

	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)

	data := RecreateCoinData{
		Name:                 "TEST",
		Symbol:               getTestCoinSymbol(),
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
	}

	tx, err := makeTestRecreateCoinTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	err = cState.Coins.Commit()
	if err != nil {
		t.Fatalf("Commit coins failed. Error %s", err)
	}

	targetBalance, _ := big.NewInt(0).SetString("0", 10)
	balance := cState.Accounts.GetBalance(addr, gasCoin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %d balance is not correct. Expected %s, got %s", gasCoin, targetBalance, balance)
	}

	targetBalance = helpers.BipToPip(big.NewInt(100))
	balance = cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %d balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	newCoinSymbol := getTestCoinSymbol()
	stateCoin := cState.Coins.GetCoinBySymbol(newCoinSymbol, 0)

	if stateCoin == nil {
		t.Fatalf("Coin %s not found in state", newCoinSymbol)
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

	if stateCoin.Version() != 0 {
		t.Fatalf("Version in state is not correct. Expected %d, got %d", 0, stateCoin.Version())
	}

	if stateCoin.Name() != "TEST" {
		t.Fatalf("Name in state is not correct. Expected TEST, got %s", stateCoin.Name())
	}

	stateCoin = cState.Coins.GetCoinBySymbol(newCoinSymbol, 1)

	if stateCoin == nil {
		t.Fatalf("Coin %s not found in state", newCoinSymbol)
	}

	if stateCoin.Version() != 1 {
		t.Fatalf("Version in state is not correct. Expected %d, got %d", 1, stateCoin.Version())
	}
}

func TestRecreateCoinTxWithWrongOwner(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	privateKey2, _ := crypto.GenerateKey()

	createTestCoinWithOwner(cState, addr)
	gasCoin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, gasCoin, helpers.BipToPip(big.NewInt(20000)))

	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)

	data := RecreateCoinData{
		Symbol:               getTestCoinSymbol(),
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
	}

	tx, err := makeTestRecreateCoinTx(data, privateKey2)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.IsNotOwnerOfCoin {
		t.Fatalf("Response code is not 206. Error %s", response.Log)
	}
}

func TestRecreateCoinTxWithWrongSymbol(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	createTestCoinWithOwner(cState, addr)
	gasCoin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, gasCoin, helpers.BipToPip(big.NewInt(20000)))

	reserve := helpers.BipToPip(big.NewInt(10000))
	amount := helpers.BipToPip(big.NewInt(100))
	crr := uint32(50)

	data := RecreateCoinData{
		Symbol:               types.StrToCoinSymbol("UNKNOWN"),
		InitialAmount:        amount,
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            big.NewInt(0).Mul(amount, big.NewInt(10)),
	}

	tx, err := makeTestRecreateCoinTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not 102. Error %s", response.Log)
	}
}

func TestRecreateCoinWithIncorrectName(t *testing.T) {
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

	data := RecreateCoinData{
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
		Type:          TypeRecreateCoin,
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
	if response.Code != code.InvalidCoinName {
		t.Fatalf("Response code is not %d. Error %s", code.InvalidCoinName, response.Log)
	}
}

func TestRecreateCoinWithWrongCrr(t *testing.T) {
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

	data := RecreateCoinData{
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
		Type:          TypeRecreateCoin,
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
	if response.Code != code.WrongCrr {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCrr, response.Log)
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
		Type:          TypeRecreateCoin,
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

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.WrongCrr {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCrr, response.Log)
	}
}

func TestRecreateCoinWithWrongCoinSupply(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("ABCDEF")
	reserve := helpers.BipToPip(big.NewInt(10000))
	crr := uint32(50)
	name := "My Test Coin"

	data := RecreateCoinData{
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
		Type:          TypeRecreateCoin,
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
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
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
		Type:          TypeRecreateCoin,
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

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
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

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
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

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.WrongCoinSupply {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCoinSupply, response.Log)
	}
}

func TestRecreateCoinWithInsufficientFundsForGas(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	createTestCoinWithOwner(cState, crypto.PubkeyToAddress(privateKey.PublicKey))
	cState.Commit()

	coin := types.GetBaseCoinID()

	toCreate := types.StrToCoinSymbol("TEST")
	reserve := helpers.BipToPip(big.NewInt(10000))
	crr := uint32(50)
	name := "My Test Coin"

	data := RecreateCoinData{
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
		Type:          TypeRecreateCoin,
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

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.SetBalance(addr, types.GetBaseCoinID(), data.InitialReserve)
	cState.Commit()

	tx.GasCoin = types.GetBaseCoinID()
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
}

func TestRecreateCoinToInsufficientFundsForInitialReserve(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	createTestCoinWithOwner(cState, crypto.PubkeyToAddress(privateKey.PublicKey))

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.SetBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(10000)))
	cState.Commit()

	toCreate := types.StrToCoinSymbol("TEST")
	reserve := helpers.BipToPip(big.NewInt(100000))
	crr := uint32(50)
	name := "My Test Coin"

	data := RecreateCoinData{
		Name:                 name,
		Symbol:               toCreate,
		InitialAmount:        helpers.BipToPip(big.NewInt(10)),
		InitialReserve:       reserve,
		ConstantReserveRatio: crr,
		MaxSupply:            helpers.BipToPip(big.NewInt(100)),
	}

	encodedTx, err := makeTestRecreateCoinTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}
}

func TestRecreateCoinToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	coin := createTestCoinWithOwner(cState, crypto.PubkeyToAddress(privateKey.PublicKey))

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(105)))
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(10000)))
	cState.Commit()

	toCreate := types.StrToCoinSymbol("TEST")
	reserve := helpers.BipToPip(big.NewInt(100000))
	crr := uint32(50)
	name := "My Test Coin"

	data := RecreateCoinData{
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
		Type:          TypeRecreateCoin,
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
	if response.Code != code.CoinReserveUnderflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveUnderflow, response.Log)
	}
}

func makeTestRecreateCoinTx(data RecreateCoinData, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		return nil, err
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeRecreateCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		return nil, err
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}

	return encodedTx, nil
}
