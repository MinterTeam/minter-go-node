package transaction

import (
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestSwitchCandidateStatusTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)

	data := SetCandidateOnData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOnline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999999900000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	candidate := cState.Candidates.GetCandidate(pubkey)

	if candidate == nil {
		t.Fatalf("Candidate not found")
	}

	if candidate.Status != candidates.CandidateStatusOnline {
		t.Fatalf("Status has not changed")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSetCandidateOffTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)

	data := SetCandidateOffData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOffline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999999900000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	candidate := cState.Candidates.GetCandidate(pubkey)

	if candidate == nil {
		t.Fatalf("Candidate not found")
	}

	if candidate.Status != candidates.CandidateStatusOffline {
		t.Fatalf("Status has not changed")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSwitchCandidateStatusTxToNonExistCandidate(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])

	data := SetCandidateOnData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOnline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CandidateNotFound {
		t.Fatalf("Response code is not %d. Error %s", code.CandidateNotFound, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSwitchCandidateStatusTxToCandidateOwnership(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	addr2 := types.Address{0}
	coin := types.GetBaseCoinID()
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr2, addr2, addr2, pubkey, 10, 0)

	data := SetCandidateOnData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOnline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.IsNotOwnerOfCandidate {
		t.Fatalf("Response code is not %d. Error %s", code.IsNotOwnerOfCandidate, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSwitchCandidateStatusToGasCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := createTestCoin(cState)
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)

	data := SetCandidateOnData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOnline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSwitchCandidateStatusToInsufficientFundsForGas(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])
	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)

	data := SetCandidateOnData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOnline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	cState.Candidates.SetOnline(pubkey)

	tx.Type = TypeSetCandidateOffline
	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSwitchCandidateStatusToCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)

	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])
	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Candidates.SetOnline(pubkey)
	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(100000)))

	data := SetCandidateOffData{
		PubKey: pubkey,
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
		Type:          TypeSetCandidateOffline,
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

	response := RunTx(cState, encodedTx, nil, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
