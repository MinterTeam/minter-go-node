package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func TestSetHaltBlockTx(t *testing.T) {
	cState := getState()

	haltHeight := uint64(100)
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
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
	if haltBlock.CandidateKey != pubkey {
		t.Fatalf("Wront halt block pubkey. Expected pubkey: %s, got %s", pubkey, haltBlock.CandidateKey.String()+"asd")
	}
}
