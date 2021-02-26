package transaction

import (
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"

	"math/big"
	"math/rand"
	"sync"
	"testing"

	db "github.com/tendermint/tm-db"
)

func TestSetHaltBlockTx(t *testing.T) {
	t.Parallel()
	cState, err := state.NewState(500000, db.NewMemDB(), nil, 1, 1, 500000)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := 500000 + uint64(100)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100)))

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

	response := RunTx(cState, encodedTx, big.NewInt(0), 500000, &sync.Map{}, 0)
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

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetHaltBlockTxWithWrongHeight(t *testing.T) {
	t.Parallel()
	currentHeight := uint64(500000 + 5)
	cState, err := state.NewState(currentHeight, db.NewMemDB(), nil, 1, 1, currentHeight)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := currentHeight - 1
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
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

	response := RunTx(cState, encodedTx, big.NewInt(0), currentHeight, &sync.Map{}, 0)
	if response.Code != code.VoiceExpired {
		t.Fatalf("Response code is not %d", code.VoiceExpired)
	}

	halts := cState.Halts.GetHaltBlocks(haltHeight)
	if halts != nil {
		t.Fatalf("Halts found at height: %d", haltHeight)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetHaltBlockTxWithWrongOwnership(t *testing.T) {
	t.Parallel()
	currentHeight := uint64(500000 + 5)
	cState, err := state.NewState(currentHeight, db.NewMemDB(), nil, 1, 1, currentHeight)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := currentHeight + 1
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
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

	response := RunTx(cState, encodedTx, big.NewInt(0), currentHeight, &sync.Map{}, 0)
	if response.Code != code.IsNotOwnerOfCandidate {
		t.Fatalf("Response code is not %d", code.IsNotOwnerOfCandidate)
	}

	halts := cState.Halts.GetHaltBlocks(haltHeight)
	if halts != nil {
		t.Fatalf("Halts found at height: %d", haltHeight)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetHaltBlockTxToNonExistCandidate(t *testing.T) {
	t.Parallel()
	cState, err := state.NewState(500000, db.NewMemDB(), nil, 1, 1, 500000)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	createDefaultValidator(cState)

	haltHeight := 500000 + uint64(100)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

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

	response := RunTx(cState, encodedTx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.CandidateNotFound {
		t.Fatalf("Response code is not %d. Error %s", code.CandidateNotFound, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetHaltBlockTxToInsufficientFunds(t *testing.T) {
	t.Parallel()
	cState, err := state.NewState(500000, db.NewMemDB(), nil, 1, 1, 500000)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := 500000 + uint64(100)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

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

	response := RunTx(cState, encodedTx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetHaltBlockTxToGasCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState, err := state.NewState(500000, db.NewMemDB(), nil, 1, 1, 500000)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	createDefaultValidator(cState)

	haltHeight := 500000 + uint64(100)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := createTestCoin(cState)
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Commit()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

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

	response := RunTx(cState, encodedTx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetHaltBlockTxToAlreadyExistenHalt(t *testing.T) {
	t.Parallel()
	cState, err := state.NewState(500000, db.NewMemDB(), nil, 1, 1, 500000)
	if err != nil {
		t.Fatalf("Cannot load state. Error %s", err)
	}

	haltHeight := 500000 + uint64(100)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1)))
	cState.Halts.AddHaltBlock(haltHeight, pubkey)

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

	response := RunTx(cState, encodedTx, big.NewInt(0), 500000, &sync.Map{}, 0)
	if response.Code != code.HaltAlreadyExists {
		t.Fatalf("response code is not %d. Error %s", code.HaltAlreadyExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
