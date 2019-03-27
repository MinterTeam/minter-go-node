package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func TestTooLongTx(t *testing.T) {
	fakeTx := make([]byte, 10000)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.TxTooLarge {
		t.Fatalf("Response code is not correct")
	}
}

func TestIncorrectTx(t *testing.T) {
	fakeTx := make([]byte, 1)
	rand.Read(fakeTx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

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

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

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

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

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

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.WrongNonce {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.WrongNonce, response.Code)
	}
}

func TestInvalidSigTx(t *testing.T) {
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
		SignatureType: SigTypeSingle,
	}

	pkey, _ := crypto.GenerateKey()

	err := tx.Sign(pkey)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	sign := [65]byte{1, 2, 3}
	tx.SetSignature(sign[:])

	fakeTx, _ := rlp.EncodeToBytes(tx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.DecodeError, response.Code)
	}
}

func TestNotExistMultiSigTx(t *testing.T) {
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
		SignatureType: SigTypeMulti,
	}

	pkey, _ := crypto.GenerateKey()
	err := tx.Sign(pkey)

	tx.multisig.Multisig = types.Address{}

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	sign := [65]byte{1, 2, 3}
	tx.SetSignature(sign[:])

	fakeTx, _ := rlp.EncodeToBytes(tx)

	response := RunTx(getState(), false, fakeTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.MultisigNotExists {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.MultisigNotExists, response.Code)
	}
}

func TestMultiSigTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	msigAddress := cState.CreateMultisig([]uint{1}, []types.Address{addr}, 1)
	cState.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

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
		SignatureType: SigTypeMulti,
	}

	err := tx.Sign(privateKey)

	tx.SetMultisigAddress(msigAddress)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	txBytes, _ := rlp.EncodeToBytes(tx)

	response := RunTx(cState, false, txBytes, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != 0 {
		t.Fatalf("Error code is not 0. Error: %s", response.Log)
	}
}

func TestMultiSigDoubleSignTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	msigAddress := cState.CreateMultisig([]uint{1, 1}, []types.Address{addr, {}}, 2)
	cState.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

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
		SignatureType: SigTypeMulti,
	}

	err := tx.Sign(privateKey)
	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}
	err = tx.Sign(privateKey)

	tx.SetMultisigAddress(msigAddress)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	txBytes, _ := rlp.EncodeToBytes(tx)

	response := RunTx(cState, false, txBytes, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.IncorrectMultiSignature {
		t.Fatalf("Error code is not %d, got %d", code.IncorrectMultiSignature, response.Code)
	}
}

func TestMultiSigTooManySignsTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	msigAddress := cState.CreateMultisig([]uint{1, 1}, []types.Address{addr, {}}, 2)
	cState.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

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
		SignatureType: SigTypeMulti,
	}

	err := tx.Sign(privateKey)
	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}
	err = tx.Sign(privateKey)
	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}
	err = tx.Sign(privateKey)
	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	tx.SetMultisigAddress(msigAddress)

	txBytes, _ := rlp.EncodeToBytes(tx)

	response := RunTx(cState, false, txBytes, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.IncorrectMultiSignature {
		t.Fatalf("Error code is not %d, got %d", code.IncorrectMultiSignature, response.Code)
	}
}

func TestMultiSigNotEnoughTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	msigAddress := cState.CreateMultisig([]uint{1}, []types.Address{addr}, 2)
	cState.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

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
		SignatureType: SigTypeMulti,
	}

	err := tx.Sign(privateKey)

	tx.SetMultisigAddress(msigAddress)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	txBytes, _ := rlp.EncodeToBytes(tx)

	response := RunTx(cState, false, txBytes, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.IncorrectMultiSignature {
		t.Fatalf("Error code is not %d. Error: %d", code.IncorrectMultiSignature, response.Code)
	}
}

func TestMultiSigIncorrectSignsTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	msigAddress := cState.CreateMultisig([]uint{1}, []types.Address{addr}, 1)
	cState.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

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
		SignatureType: SigTypeMulti,
	}

	err := tx.Sign(privateKey)
	tx.multisig.Signatures[0].S = types.Big0

	tx.SetMultisigAddress(msigAddress)

	if err != nil {
		t.Fatalf("Error %s", err.Error())
	}

	txBytes, _ := rlp.EncodeToBytes(tx)

	response := RunTx(cState, false, txBytes, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != code.IncorrectMultiSignature {
		t.Fatalf("Error code is not %d, got %d", code.IncorrectMultiSignature, response.Code)
	}
}
