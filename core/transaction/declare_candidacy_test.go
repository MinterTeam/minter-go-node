package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"

	"math/big"
	"sync"
	"testing"
)

func TestDeclareCandidacyTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	commission := uint32(10)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: commission,
		Coin:       coin,
		Stake:      helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeDeclareCandidacy,
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

	targetBalance, _ := big.NewInt(0).SetString("999890000000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	candidate := cState.Candidates.GetCandidate(publicKey)

	if candidate == nil {
		t.Fatalf("Candidate not found")
	}

	if candidate.OwnerAddress != addr {
		t.Fatalf("Owner address is not correct")
	}

	if candidate.RewardAddress != addr {
		t.Fatalf("Reward address is not correct")
	}

	if candidate.ControlAddress != addr {
		t.Fatalf("Control address is not correct")
	}

	if candidate.GetTotalBipStake() != nil && candidate.GetTotalBipStake().Cmp(types.Big0) != 0 {
		t.Fatalf("Total stake is not correct")
	}

	if candidate.Commission != commission {
		t.Fatalf("Commission is not correct")
	}

	if candidate.Status != candidates.CandidateStatusOffline {
		t.Fatalf("Incorrect candidate status")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyTxOverflow(t *testing.T) {
	cState := getState()
	maxCandidatesCount := validators.GetCandidatesCountForBlock(0)

	for i := 0; i < maxCandidatesCount; i++ {
		pubkey := types.Pubkey{byte(i)}
		cState.Candidates.Create(types.Address{}, types.Address{}, types.Address{}, pubkey, 10)
		cState.Candidates.Delegate(types.Address{}, pubkey, types.GetBaseCoinID(), helpers.BipToPip(big.NewInt(10)), helpers.BipToPip(big.NewInt(10)))
	}

	cState.Candidates.RecalculateStakes(109000)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: uint32(10),
		Coin:       coin,
		Stake:      helpers.BipToPip(big.NewInt(10)),
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
		Type:          TypeDeclareCandidacy,
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

	if response.Code != code.TooLowStake {
		t.Fatalf("Response code is not %d. Got %d", code.TooLowStake, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyTxWithBlockPybKey(t *testing.T) {
	cState := getState()

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	cState.Candidates.Create(types.Address{}, types.Address{}, types.Address{}, publicKey, 10)
	pkeyNew, _ := crypto.GenerateKey()
	publicKeyNewBytes := crypto.FromECDSAPub(&pkeyNew.PublicKey)[:32]
	var publicKeyNew types.Pubkey
	copy(publicKeyNew[:], publicKeyNewBytes)

	cState.Candidates.ChangePubKey(publicKey, publicKeyNew)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	commission := uint32(10)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: commission,
		Coin:       coin,
		Stake:      helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeDeclareCandidacy,
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

	if response.Code == 0 {
		t.Fatal("Response code is not 1. Want error")
	}

	if cState.Candidates.GetCandidate(publicKey) != nil {
		t.Fatalf("Candidate found with old pub key")
	}

	candidate := cState.Candidates.GetCandidate(publicKeyNew)

	if candidate == nil {
		t.Fatalf("Candidate not found")
	}

	if candidate.OwnerAddress == addr {
		t.Fatalf("OwnerAddress has changed")
	}

	if candidate.RewardAddress == addr {
		t.Fatalf("RewardAddress has changed")
	}

	if candidate.ControlAddress == addr {
		t.Fatalf("ControlAddress has changed")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyToNonExistCoin(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	commission := uint32(10)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: commission,
		Coin:       5,
		Stake:      helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeDeclareCandidacy,
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

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyToExistCandidate(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	cState.Candidates.Create(addr, addr, addr, publicKey, uint32(10))

	commission := uint32(10)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: commission,
		Coin:       coin,
		Stake:      helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeDeclareCandidacy,
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
		t.Fatalf("Response code is not %d. Error %s", code.CandidateExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyToDecodeError(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	commission := uint32(10)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: commission,
		Coin:       5,
		Stake:      nil,
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
		Type:          TypeDeclareCandidacy,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	response := data.Run(&tx, state.NewCheckState(cState), nil, 0)
	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not %d. Error %s", code.DecodeError, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyToWrongCommission(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: maxCommission + 1,
		Coin:       coin,
		Stake:      helpers.BipToPip(big.NewInt(100)),
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
		Type:          TypeDeclareCandidacy,
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
	if response.Code != code.WrongCommission {
		t.Fatalf("Response code is not %d. Error %s", code.WrongCommission, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyToInsufficientFunds(t *testing.T) {
	cState := getState()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	stake := big.NewInt(1e18)

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: uint32(10),
		Coin:       coin,
		Stake:      stake,
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
		Type:          TypeDeclareCandidacy,
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

	cState.Accounts.AddBalance(addr, coin, stake)
	cState.Commit()

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
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	cState.Accounts.SetBalance(addr, coin, new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10)))
	cState.Commit()

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
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestDeclareCandidacyTxToGasCoinReserveUnderflow(t *testing.T) {
	cState := getState()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)
	stake := big.NewInt(1e18)

	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Commit()

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: uint32(10),
		Coin:       coin,
		Stake:      stake,
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
		Type:          TypeDeclareCandidacy,
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

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
