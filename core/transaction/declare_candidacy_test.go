package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
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

	coin := types.GetBaseCoin()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	commission := uint(10)

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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

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

	if candidate.GetTotalBipStake() != nil && candidate.GetTotalBipStake().Cmp(types.Big0) != 0 {
		t.Fatalf("Total stake is not correct")
	}

	if candidate.Commission != commission {
		t.Fatalf("Commission is not correct")
	}

	if candidate.Status != candidates.CandidateStatusOffline {
		t.Fatalf("Incorrect candidate status")
	}
}

func TestDeclareCandidacyTxOverflow(t *testing.T) {
	cState := getState()
	maxCandidatesCount := validators.GetCandidatesCountForBlock(0)

	for i := 0; i < maxCandidatesCount; i++ {
		pubkey := types.Pubkey{byte(i)}
		cState.Candidates.Create(types.Address{}, types.Address{}, pubkey, 10)
		cState.Candidates.Delegate(types.Address{}, pubkey, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(10)), helpers.BipToPip(big.NewInt(10)))
	}

	cState.Candidates.RecalculateStakes(0)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoin()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	pkey, _ := crypto.GenerateKey()
	publicKeyBytes := crypto.FromECDSAPub(&pkey.PublicKey)[:32]
	var publicKey types.Pubkey
	copy(publicKey[:], publicKeyBytes)

	data := DeclareCandidacyData{
		Address:    addr,
		PubKey:     publicKey,
		Commission: uint(10),
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, 0)

	if response.Code != code.TooLowStake {
		t.Fatalf("Response code is not %d. Got %d", code.TooLowStake, response.Code)
	}
}
