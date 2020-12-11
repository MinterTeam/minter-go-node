package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func TestEditCandidateNewPublicKeyTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := types.HexToPubkey("Mp11fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c611")
	newpubkey := types.HexToPubkey("Mp12fdfc072182654f163f5f0f9a621d729566c74d10037c4d7bbb0407d1e2c612")

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Candidates.SetStakes(pubkey, []types.Stake{
		{
			Owner:    addr,
			Coin:     0,
			Value:    "1000000000000000000000",
			BipValue: "1000000000000000000000",
		},
	}, nil)
	cState.Candidates.SetOnline(pubkey)
	cState.Candidates.RecalculateStakes(0)
	cState.Validators.SetNewValidators(cState.Candidates.GetNewCandidates(1))

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: newpubkey,
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
		Type:          TypeEditCandidatePublicKey,
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

	targetBalance, _ := big.NewInt(0).SetString("900000000000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	if cState.Candidates.GetCandidate(pubkey) != nil {
		t.Fatalf("Candidate found with old pub key")
	}

	candidate := cState.Candidates.GetCandidate(newpubkey)
	if candidate == nil {
		t.Fatalf("Candidate not found")
	}

	if !candidate.PubKey.Equals(newpubkey) {
		t.Fatalf("Public key has not changed")
	}

	cState.Validators.SetNewValidators(cState.Candidates.GetNewCandidates(1))
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCandidatePublicKeyTxToNewPublicKey(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: pubkey,
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
		Type:          TypeEditCandidatePublicKey,
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
	if response.Code != code.NewPublicKeyIsBad {
		t.Fatalf("Response code is not %d. Error %s", code.NewPublicKeyIsBad, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCandidatePublicKeyTxToNewPublicKeyInBlockList(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	pubkey2 := types.Pubkey{1}

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	cState.Candidates.AddToBlockPubKey(pubkey2)

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: pubkey2,
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
		Type:          TypeEditCandidatePublicKey,
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
	if response.Code != code.PublicKeyInBlockList {
		t.Fatalf("Response code is not %d. Error %s", code.PublicKeyInBlockList, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCandidatePublicKeyTxToInsufficientFunds(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: types.Pubkey{5},
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
		Type:          TypeEditCandidatePublicKey,
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

func TestEditCandidatePublicKeyTxToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: types.Pubkey{5},
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
		Type:          TypeEditCandidatePublicKey,
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
	if response.Code != code.CoinReserveNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CoinReserveNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCandidatePublicKeyToNotExistCandidate(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: types.Pubkey{5},
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
		Type:          TypeEditCandidatePublicKey,
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
	if response.Code != code.CandidateNotFound {
		t.Fatalf("Response code is not %d. Error %s", code.CandidateNotFound, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCandidatePublicKeyTxToCandidateOwnership(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	addr2 := types.Address{0}
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr2, addr2, addr2, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: types.Pubkey{5},
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
		Type:          TypeEditCandidatePublicKey,
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
	if response.Code != code.IsNotOwnerOfCandidate {
		t.Fatalf("Response code is not %d. Error %s", code.IsNotOwnerOfCandidate, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestEditCandidatePublicKeyData_Exists(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])
	newpubkey := [32]byte{}
	rand.Read(newpubkey[:])

	cState.Candidates.Create(addr, addr, addr, newpubkey, 10)
	cState.Validators.Create(newpubkey, helpers.BipToPip(big.NewInt(1)))

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	candidate1 := cState.Candidates.GetCandidate(newpubkey)
	if candidate1 == nil {
		t.Fatalf("Candidate not found")
	}
	candidate2 := cState.Candidates.GetCandidate(pubkey)
	if candidate2 == nil {
		t.Fatalf("Candidate not found")
	}

	data := EditCandidatePublicKeyData{
		PubKey:    pubkey,
		NewPubKey: newpubkey,
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
		Type:          TypeEditCandidatePublicKey,
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
	if response.Code != code.CandidateExists {
		t.Fatalf("Response code is %d. Error %s", response.Code, response.Log)
	}

	if candidate1.PubKey == candidate2.PubKey {
		t.Fatalf("Candidates pulic keys are equal")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
