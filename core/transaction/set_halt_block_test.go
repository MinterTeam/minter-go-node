package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/upgrades"
	db "github.com/tendermint/tm-db"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func TestSetHaltBlockTx(t *testing.T) {
	cState, err := state.NewState(upgrades.UpgradeBlock4, db.NewMemDB(), nil, 1, 1)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := upgrades.UpgradeBlock4 + uint64(100)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	data := SetHaltBlockData{
		PubKey: pubkey,
		Height: haltHeight,
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
		Type:          TypeSetHaltBlock,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), upgrades.UpgradeBlock4, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("0", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	halts := cState.Halts.GetHaltBlocks(haltHeight)
	if halts == nil {
		t.Fatalf("No halts on the height: %d", haltHeight)
	}

	haltBlocks := halts.List
	if len(haltBlocks) != 1 {
		t.Fatalf("Halt blocks are not correct. Expected halts size: %d, got %d", 1, len(haltBlocks))
	}

	haltBlock := haltBlocks[0]
	if haltBlock.Pubkey != pubkey {
		t.Fatalf("Wront halt block pubkey. Expected pubkey: %s, got %s", pubkey, haltBlock.Pubkey.String()+"asd")
	}
}

func TestSetHaltBlockTxWithWrongHeight(t *testing.T) {
	currentHeight := uint64(upgrades.UpgradeBlock4 + 5)
	cState, err := state.NewState(currentHeight, db.NewMemDB(), nil, 1, 1)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := currentHeight - 1
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	data := SetHaltBlockData{
		PubKey: pubkey,
		Height: haltHeight,
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
		Type:          TypeSetHaltBlock,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), currentHeight, &sync.Map{}, 0)
	if response.Code != code.WrongHaltHeight {
		t.Fatalf("Response code is not %d", code.WrongHaltHeight)
	}

	halts := cState.Halts.GetHaltBlocks(haltHeight)
	if halts != nil {
		t.Fatalf("Halts found at height: %d", haltHeight)
	}
}

func TestSetHaltBlockTxWithWrongOwnership(t *testing.T) {
	currentHeight := uint64(upgrades.UpgradeBlock4 + 5)
	cState, err := state.NewState(currentHeight, db.NewMemDB(), nil, 1, 1)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := currentHeight + 1
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))

	data := SetHaltBlockData{
		PubKey: pubkey,
		Height: haltHeight,
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
		Type:          TypeSetHaltBlock,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	privateKey2, _ := crypto.GenerateKey()
	if err := tx.Sign(privateKey2); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, false, encodedTx, big.NewInt(0), currentHeight, &sync.Map{}, 0)
	if response.Code != code.IsNotOwnerOfCandidate {
		t.Fatalf("Response code is not %d", code.IsNotOwnerOfCandidate)
	}

	halts := cState.Halts.GetHaltBlocks(haltHeight)
	if halts != nil {
		t.Fatalf("Halts found at height: %d", haltHeight)
	}
}
