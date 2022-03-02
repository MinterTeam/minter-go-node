package transaction

import (
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestLockStakeTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	data := LockStakeData{}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeLockStake,
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

	response := NewExecutor(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
