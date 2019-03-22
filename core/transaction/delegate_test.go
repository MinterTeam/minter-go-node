package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func createTestCandidate(stateDB *state.StateDB) []byte {
	address := types.Address{}
	pubkey := make([]byte, 32)
	rand.Read(pubkey)

	stateDB.CreateCandidate(address, address, pubkey, 10, 0, types.GetBaseCoin(), helpers.BipToPip(big.NewInt(1)))

	return pubkey
}

func TestDelegateTx(t *testing.T) {
	cState := getState()

	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoin()

	cState.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(100))

	data := DelegateData{
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
		GasPrice:      big.NewInt(1),
		GasCoin:       coin,
		Type:          TypeDelegate,
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999899800000000000000000", 10)
	balance := cState.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	candidate := cState.GetStateCandidate(pubkey)

	stake := candidate.GetStakeOfAddress(addr, coin)

	if stake == nil {
		t.Fatalf("Stake not found")
	}

	if stake.Value.Cmp(value) != 0 {
		t.Fatalf("Stake value is not corrent. Expected %s, got %s", value, stake.Value)
	}
}
