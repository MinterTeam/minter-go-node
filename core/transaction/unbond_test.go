package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"testing"
)

func TestUnbondTx(t *testing.T) {
	cState := getState()

	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	cState.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(100))
	cState.Delegate(addr, pubkey, coin, value)

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
		GasPrice:      big.NewInt(1),
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, make(map[types.Address]struct{}))

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999999800000000000000000", 10)
	balance := cState.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	candidate := cState.GetStateCandidate(pubkey)

	stake := candidate.GetStakeOfAddress(addr, coin)

	if stake.Value.Cmp(types.Big0) != 0 {
		t.Fatalf("Stake value is not corrent. Expected %s, got %s", types.Big0, stake.Value)
	}
}
