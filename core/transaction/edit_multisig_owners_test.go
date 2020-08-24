package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"reflect"
	"sync"
	"testing"
)

func TestEditMultisigOwnersTx(t *testing.T) {
	cState := getState()

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	addr := types.Address{0}
	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)
	privateKey3, _ := crypto.GenerateKey()
	addr3 := crypto.PubkeyToAddress(privateKey3.PublicKey)
	privateKey4, _ := crypto.GenerateKey()
	addr4 := crypto.PubkeyToAddress(privateKey4.PublicKey)

	cState.Accounts.CreateMultisig([]uint{1, 2, 3}, []types.Address{addr1, addr2, addr3}, 3, 1, addr)

	coin := types.GetBaseCoinID()
	initialBalance := big.NewInt(1)
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(initialBalance))

	data := EditMultisigOwnersData{
		Threshold:       3,
		Weights:         []uint{2, 1, 2},
		Addresses:       []types.Address{addr1, addr2, addr4},
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
		Type:          TypeEditMultisigOwner,
		Data:          encodedData,
		SignatureType: SigTypeMulti,
	}

	tx.SetMultisigAddress(addr)

	if err := tx.Sign(privateKey3); err != nil {
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

	account := cState.Accounts.GetAccount(addr)

	if !account.IsMultisig() {
		t.Fatalf("Multisig %s is not created", addr.String())
	}

	msigData := account.Multisig()

	if !reflect.DeepEqual(msigData.Addresses, data.Addresses) {
		t.Fatalf("Addresses are not correct")
	}

	if !reflect.DeepEqual(msigData.Weights, data.Weights) {
		t.Fatalf("Weights are not correct")
	}

	if msigData.Threshold != 3 {
		t.Fatalf("Threshold is not correct")
	}
}
