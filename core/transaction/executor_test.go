package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"testing"
)

func TestTooLongTx(t *testing.T) {
	fakeTx := make([]byte, 2177)
	rand.Read(fakeTx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0)

	if response.Code != code.TxTooLarge {
		t.Fatalf("Response code is not correct")
	}
}

func TestIncorrectTx(t *testing.T) {
	fakeTx := make([]byte, 1)
	rand.Read(fakeTx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0)

	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not correct")
	}
}

func TestTooLongPayloadTx(t *testing.T) {
	payload := make([]byte, 1025)
	rand.Read(payload)

	txData := SendData{
		Coin:  types.GetBaseCoin(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      big.NewInt(1),
		GasCoin:       types.GetBaseCoin(),
		Type:          TypeSend,
		Data:          encodedData,
		Payload:       payload,
		ServiceData:   nil,
		SignatureType: SigTypeSingle,
	}

	pkey, _ := crypto.GenerateKey()

	err := tx.Sign(pkey)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	fakeTx, _ := rlp.EncodeToBytes(tx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0)

	if response.Code != code.TxPayloadTooLarge {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.TxPayloadTooLarge, response.Code)
	}
}

func TestTooLongServiceDataTx(t *testing.T) {
	serviceData := make([]byte, 1025)
	rand.Read(serviceData)

	txData := SendData{
		Coin:  types.GetBaseCoin(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      big.NewInt(1),
		GasCoin:       types.GetBaseCoin(),
		Type:          TypeSend,
		Data:          encodedData,
		ServiceData:   serviceData,
		SignatureType: SigTypeSingle,
	}

	pkey, _ := crypto.GenerateKey()

	err := tx.Sign(pkey)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	fakeTx, _ := rlp.EncodeToBytes(tx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0)

	if response.Code != code.TxServiceDataTooLarge {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.TxServiceDataTooLarge, response.Code)
	}
}

func TestUnexpectedNonceTx(t *testing.T) {
	txData := SendData{
		Coin:  types.GetBaseCoin(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         2,
		GasPrice:      big.NewInt(1),
		GasCoin:       types.GetBaseCoin(),
		Type:          TypeSend,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	pkey, _ := crypto.GenerateKey()

	err := tx.Sign(pkey)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	fakeTx, _ := rlp.EncodeToBytes(tx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0)

	if response.Code != code.WrongNonce {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.WrongNonce, response.Code)
	}
}
