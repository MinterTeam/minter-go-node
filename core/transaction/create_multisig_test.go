package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"reflect"
	"sync"
	"testing"
)

func TestCreateMultisigTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	coin := types.GetBaseCoin()

	cState.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	addresses := []types.Address{
		addr,
		addr2,
	}

	weights := []uint{1, 1}

	data := CreateMultisigData{
		Threshold: 1,
		Weights:   weights,
		Addresses: addresses,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      big.NewInt(1),
		GasCoin:       coin,
		Type:          TypeCreateMultisig,
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

	targetBalance, _ := big.NewInt(0).SetString("999999900000000000000000", 10)
	balance := cState.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	var multisigAddr types.Address

	for _, item := range response.Tags {
		if string(item.Key) == "tx.created_multisig" {
			multisigAddr = types.HexToAddress(string(item.Value))
		}
	}

	multisig := cState.GetOrNewStateObject(multisigAddr)

	if !multisig.IsMultisig() {
		t.Fatalf("Multisig %s is not created", multisigAddr.String())
	}

	msigData := multisig.Multisig()

	if !reflect.DeepEqual(msigData.Addresses, addresses) {
		t.Fatalf("Addresses are not correct")
	}

	if !reflect.DeepEqual(msigData.Weights, weights) {
		t.Fatalf("Weights are not correct")
	}

	if msigData.Threshold != 1 {
		t.Fatalf("Threshold is not correct")
	}
}
