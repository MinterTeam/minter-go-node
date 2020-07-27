package transaction

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sync"
	"testing"
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
	crr := uint(50)

	data := RecreateCoinData{
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
	stateCoin := cState.Coins.GetCoinBySymbol(newCoinSymbol)

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

	archiveCoinSymbol := types.StrToCoinSymbol(fmt.Sprintf("%s-1", getTestCoinSymbol()))
	stateCoin = cState.Coins.GetCoinBySymbol(archiveCoinSymbol)

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
	crr := uint(50)

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
	crr := uint(50)

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