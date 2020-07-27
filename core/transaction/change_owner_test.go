package transaction

import (
	"crypto/ecdsa"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/upgrades"
	db "github.com/tendermint/tm-db"
	"math/big"
	"sync"
	"testing"
)

func TestChangeOwnerTx(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, addr)

	gasCoin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, gasCoin, helpers.BipToPip(big.NewInt(10000)))

	data := ChangeOwnerData{
		Symbol:   getTestCoinSymbol(),
		NewOwner: newOwner,
	}

	tx, err := makeTestChangeOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), upgrades.UpgradeBlock4, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	err = cState.Coins.Commit()
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
}

func TestChangeOwnerTxWithWrongOwner(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, newOwner)

	data := ChangeOwnerData{
		Symbol:   getTestCoinSymbol(),
		NewOwner: newOwner,
	}

	tx, err := makeTestChangeOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), upgrades.UpgradeBlock4, &sync.Map{}, 0)
	if response.Code != code.IsNotOwnerOfCoin {
		t.Fatalf("Response code is not 206. Error %s", response.Log)
	}
}

func TestChangeOwnerTxWithWrongSymbol(t *testing.T) {
	cState, err := state.NewState(0, db.NewMemDB(), nil, 1, 1)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	newOwnerPrivateKey, _ := crypto.GenerateKey()
	newOwner := crypto.PubkeyToAddress(newOwnerPrivateKey.PublicKey)

	createTestCoinWithOwner(cState, addr)

	data := ChangeOwnerData{
		Symbol:   types.StrToCoinSymbol("UNKNOWN"),
		NewOwner: newOwner,
	}

	tx, err := makeTestChangeOwnerTx(data, privateKey)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, tx, big.NewInt(0), upgrades.UpgradeBlock4, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not 102. Error %s", response.Log)
	}
}

func makeTestChangeOwnerTx(data ChangeOwnerData, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		return nil, err
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeChangeOwner,
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