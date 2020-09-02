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

func TestEditCandidateTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	data := EditCandidateData{
		PubKey:         pubkey,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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

	targetBalance, _ := big.NewInt(0).SetString("999990000000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	candidate := cState.Candidates.GetCandidate(pubkey)

	if candidate == nil {
		t.Fatalf("Candidate not found")
	}

	if candidate.OwnerAddress != newOwnerAddress {
		t.Fatalf("OwnerAddress has not changed")
	}

	if candidate.RewardAddress != newRewardAddress {
		t.Fatalf("RewardAddress has not changed")
	}

	if candidate.ControlAddress != newControlAddress {
		t.Fatalf("ControlAddress has not changed")
	}
}

func TestEditCandidateNewPubKeyTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])
	newpubkey := [32]byte{}
	rand.Read(newpubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	p := types.Pubkey(newpubkey)
	data := EditCandidateData{
		PubKey:         pubkey,
		NewPubKey:      &p,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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

	targetBalance, _ := big.NewInt(0).SetString("999990000000000000000000", 10)
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

	if candidate.OwnerAddress != newOwnerAddress {
		t.Fatalf("OwnerAddress has not changed")
	}

	if candidate.RewardAddress != newRewardAddress {
		t.Fatalf("RewardAddress has not changed")
	}

	if candidate.ControlAddress != newControlAddress {
		t.Fatalf("ControlAddress has not changed")
	}
}

func TestEditCandidateTxToNonExistenCandidate(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	data := EditCandidateData{
		PubKey:         pubkey,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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
}

func TestEditCandidateTxToCandidateOwnership(t *testing.T) {
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

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	data := EditCandidateData{
		PubKey:         pubkey,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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
}

func TestEditCandidateTxToNewPublicKey(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}
	newPubKey := types.Pubkey(pubkey)

	data := EditCandidateData{
		PubKey:         pubkey,
		NewPubKey:      &newPubKey,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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
}

func TestEditCandidateTxToInsufficientFunds(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	data := EditCandidateData{
		PubKey:         pubkey,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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
}

func TestEditCandidateTxToNewPublicKeyInBlockList(t *testing.T) {
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

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	data := EditCandidateData{
		PubKey:         pubkey,
		NewPubKey:      &pubkey2,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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
}

func TestEditCandidateTxToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	pubkey2 := types.Pubkey{1}

	cState.Candidates.Create(addr, addr, addr, pubkey, 10)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))

	newRewardAddress := types.Address{1}
	newOwnerAddress := types.Address{2}
	newControlAddress := types.Address{3}

	data := EditCandidateData{
		PubKey:         pubkey,
		NewPubKey:      &pubkey2,
		RewardAddress:  newRewardAddress,
		OwnerAddress:   newOwnerAddress,
		ControlAddress: newControlAddress,
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
		Type:          TypeEditCandidate,
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
