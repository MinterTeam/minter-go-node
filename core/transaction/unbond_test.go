package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"

	"math/big"
	"sync"
	"testing"
)

func TestUnbondTx(t *testing.T) {
	cState := getState()

	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(100))
	cState.Candidates.Delegate(addr, pubkey, coin, value, big.NewInt(0))

	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeUnbond,
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

	cState.Candidates.RecalculateStakes(109000)

	targetBalance, _ := big.NewInt(0).SetString("999999800000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	stake := cState.Candidates.GetStakeOfAddress(pubkey, addr, coin)

	if stake.Value.Cmp(types.Big0) != 0 {
		t.Fatalf("Stake value is not corrent. Expected %s, got %s", types.Big0, stake.Value)
	}

	checkState(t, cState)
}

func TestFullUnbondTxWithWaitlist(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	waitlistAmount := helpers.BipToPip(big.NewInt(1000))
	value := helpers.BipToPip(big.NewInt(1000))

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Waitlist.AddWaitList(addr, pubkey, coin, waitlistAmount)
	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeUnbond,
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

	cState.Candidates.RecalculateStakes(109000)
	funds := cState.FrozenFunds.GetFrozenFunds(candidates.UnbondPeriod)
	if funds == nil || len(funds.List) != 1 {
		t.Fatalf("Frozen funds are not correct")
	}

	stake := cState.Candidates.GetStakeOfAddress(pubkey, addr, coin)
	if stake != nil {
		t.Fatalf("Stake value is not empty.")
	}

	if funds.List[0].Value.Cmp(value) != 0 {
		t.Fatalf("Frozen funds value is not corrent. Expected %s, got %s", value, funds.List[0].Value)
	}

	wl := cState.Waitlist.Get(addr, pubkey, coin)
	if wl != nil {
		t.Fatalf("Waitlist is not deleted")
	}

	checkState(t, cState)
}

func TestUnbondTxWithWaitlist(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	waitlistAmount := helpers.BipToPip(big.NewInt(1000))
	unbondAmount := helpers.BipToPip(big.NewInt(50))

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Waitlist.AddWaitList(addr, pubkey, coin, waitlistAmount)
	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  unbondAmount,
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
		Type:          TypeUnbond,
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

	cState.Candidates.RecalculateStakes(109000)
	funds := cState.FrozenFunds.GetFrozenFunds(candidates.UnbondPeriod)
	if funds == nil || len(funds.List) != 1 {
		t.Fatalf("Frozen funds are not correct")
	}

	if funds.List[0].Value.Cmp(unbondAmount) != 0 {
		t.Fatalf("Frozen funds value is not corrent. Expected %s, got %s", unbondAmount, funds.List[0].Value)
	}

	wl := cState.Waitlist.Get(addr, pubkey, coin)
	if wl == nil {
		t.Fatalf("Waitlist is empty")
	}

	amount := new(big.Int).Sub(waitlistAmount, unbondAmount)
	if wl.Value.Cmp(amount) != 0 {
		t.Fatalf("Waitlist is not correct")
	}

	checkState(t, cState)
}

func TestUnbondTxToDecodeError(t *testing.T) {
	cState := getState()

	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	value := helpers.BipToPip(big.NewInt(100))
	cState.Candidates.Delegate(addr, pubkey, coin, value, big.NewInt(0))

	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
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
		Type:          TypeUnbond,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	response := data.Run(&tx, state.NewCheckState(cState, nil), nil, 1)
	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not %d. Error %s", code.DecodeError, response.Log)
	}

	checkState(t, cState)
}

func TestUnbondTxToNotExistCoin(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	data := UnbondData{
		PubKey: pubkey,
		Coin:   5,
		Value:  value,
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
		Type:          TypeUnbond,
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
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	checkState(t, cState)
}

func TestUnbondTxToNotExistCandidate(t *testing.T) {
	cState := getState()
	pubkey := types.Pubkey{1}
	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeUnbond,
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

	checkState(t, cState)
}

func TestUnbondTxToNotExistStake(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeUnbond,
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
	if response.Code != code.StakeNotFound {
		t.Fatalf("Response code is not %d. Error %s", code.StakeNotFound, response.Log)
	}

	checkState(t, cState)
}

func TestUnbondTxToInsufficientStake(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Candidates.Delegate(addr, pubkey, coin, value, big.NewInt(0))
	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  helpers.BipToPip(big.NewInt(1000)),
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
		Type:          TypeUnbond,
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
	if response.Code != code.InsufficientStake {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientStake, response.Log)
	}

	checkState(t, cState)
}

func TestUnbondTxToInsufficientFunds(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	cState.Candidates.Delegate(addr, pubkey, coin, value, big.NewInt(0))
	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeUnbond,
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

	checkState(t, cState)
}

func TestUnbondTxToInsufficientAmountAtWaitlist(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	cState.Waitlist.AddWaitList(addr, pubkey, coin, value)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  helpers.BipToPip(big.NewInt(1000)),
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
		Type:          TypeUnbond,
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
	if response.Code != code.InsufficientWaitList {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientWaitList, response.Log)
	}

	checkState(t, cState)
}

func TestUnbondTxToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))

	cState.Candidates.Delegate(addr, pubkey, coin, value, big.NewInt(0))
	cState.Candidates.RecalculateStakes(109000)

	data := UnbondData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	gasCoin := createTestCoin(cState)
	cState.Coins.SubReserve(gasCoin, helpers.BipToPip(big.NewInt(90000)))

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       gasCoin,
		Type:          TypeUnbond,
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

	checkState(t, cState)
}
