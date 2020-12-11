package transaction

import (
	"crypto/ecdsa"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"

	"math/big"
	"sync"
	"testing"

	db "github.com/tendermint/tm-db"
)

func TestEditOwnerTx(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1, 0)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, addr)
	createDefaultValidator(cState)

	gasCoin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, gasCoin, helpers.BipToPip(big.NewInt(10000)))

	data := EditCoinOwnerData{
		Symbol:   getTestCoinSymbol(),
		NewOwner: newOwner,
	}

	tx, err := makeTestEditOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	_, _, err = cState.Tree().Commit(cState.Coins)
	if err != nil {
		t.Fatalf("Failed to commit coins: %s", err)
	}

	targetBalance, _ := big.NewInt(0).SetString("0", 10)
	balance := cState.Accounts.GetBalance(addr, gasCoin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %d, got %s", gasCoin, targetBalance, balance)
	}

	symbol := cState.Coins.GetSymbolInfo(getTestCoinSymbol())
	if symbol == nil {
		t.Fatal("Symbol info not found")
	}

	if *symbol.OwnerAddress() != newOwner {
		t.Fatalf("Target owner address is not correct. Excpected %s, got %s", newOwner.String(), symbol.OwnerAddress().String())
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditOwnerTxWithWrongOwner(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1, 0)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, newOwner)
	createDefaultValidator(cState)

	data := EditCoinOwnerData{
		Symbol:   getTestCoinSymbol(),
		NewOwner: newOwner,
	}

	tx, err := makeTestEditOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.IsNotOwnerOfCoin {
		t.Fatalf("Response code is not 206. Error %s", response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditOwnerTxWithWrongSymbol(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1, 0)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, addr)
	createDefaultValidator(cState)

	data := EditCoinOwnerData{
		Symbol:   types.StrToCoinSymbol("UNKNOWN"),
		NewOwner: newOwner,
	}

	tx, err := makeTestEditOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not 102. Error %s", response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCOwnerTxWithInsufficientFunds(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1, 0)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, addr)
	createDefaultValidator(cState)

	data := EditCoinOwnerData{
		Symbol:   getTestCoinSymbol(),
		NewOwner: newOwner,
	}

	tx, err := makeTestEditOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCoinOwnerTxToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	customCoin := createTestCoinWithOwner(cState, addr)
	cState.Coins.SubReserve(customCoin, helpers.BipToPip(big.NewInt(90000)))

	data := EditCoinOwnerData{
		Symbol:   types.StrToCoinSymbol("TEST"),
		NewOwner: newOwner,
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
		Type:          TypeEditCoinOwner,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.CoinReserveUnderflow {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveUnderflow, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func makeTestEditOwnerTx(data EditCoinOwnerData, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		return nil, err
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeEditCoinOwner,
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

func createDefaultValidator(cState *state.State) {
	cState.Validators.Create(types.Pubkey{0}, big.NewInt(0))
	cState.Candidates.Create(types.Address{0}, types.Address{0}, types.Address{0}, types.Pubkey{0}, 0)
}
