package transaction

import (
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestTooLongTx(t *testing.T) {
	t.Parallel()
	fakeTx := make([]byte, maxTxLength+1)

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.TxTooLarge {
		t.Fatalf("Response code is not correct")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestIncorrectTx(t *testing.T) {
	t.Parallel()
	fakeTx := make([]byte, 1)
	rand.Read(fakeTx)

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not correct")
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestTooLongPayloadTx(t *testing.T) {
	t.Parallel()
	payload := make([]byte, maxPayloadLength+1)
	rand.Read(payload)

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
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

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.TxPayloadTooLarge {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.TxPayloadTooLarge, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestTooLongServiceDataTx(t *testing.T) {
	t.Parallel()
	serviceData := make([]byte, 1025)
	rand.Read(serviceData)

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
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

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.TxServiceDataTooLarge {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.TxServiceDataTooLarge, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestUnexpectedNonceTx(t *testing.T) {
	t.Parallel()
	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         2,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
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

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.WrongNonce {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.WrongNonce, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestInvalidSigTx(t *testing.T) {
	t.Parallel()
	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		ChainID:       types.CurrentChainID,
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

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.DecodeError, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestNotExistMultiSigTx(t *testing.T) {
	t.Parallel()
	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSend,
		ChainID:       types.CurrentChainID,
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

	cState := getState()
	response := NewExecutor(GetData).RunTx(cState, fakeTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.MultisigNotExists {
		t.Fatalf("Response code is not correct. Expected %d, got %d", code.MultisigNotExists, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultiSigTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	msigAddress := cState.Accounts.CreateMultisig([]uint32{1}, []types.Address{addr}, 1, accounts.CreateMultisigAddress(addr, 1))
	cState.Accounts.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		ChainID:       types.CurrentChainID,
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

	response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != 0 {
		t.Fatalf("Error code is not 0. Error: %s", response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultiSigDoubleSignTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	msigAddress := cState.Accounts.CreateMultisig([]uint32{1, 1}, []types.Address{addr, {}}, 2, accounts.CreateMultisigAddress(addr, 1))
	cState.Accounts.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSend,
		ChainID:       types.CurrentChainID,
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

	response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.DuplicatedAddresses {
		t.Fatalf("Error code is not %d, got %d", code.DuplicatedAddresses, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultiSigTooManySignsTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	msigAddress := cState.Accounts.CreateMultisig([]uint32{1, 1}, []types.Address{addr, {}}, 2, accounts.CreateMultisigAddress(addr, 1))
	cState.Accounts.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		ChainID:       types.CurrentChainID,
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

	response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.IncorrectMultiSignature {
		t.Fatalf("Error code is not %d, got %d", code.IncorrectMultiSignature, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultiSigNotEnoughTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	msigAddress := cState.Accounts.CreateMultisig([]uint32{1}, []types.Address{addr}, 2, accounts.CreateMultisigAddress(addr, 1))
	cState.Accounts.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
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

	response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.NotEnoughMultisigVotes {
		t.Fatalf("Error code is not %d. Error: %d", code.NotEnoughMultisigVotes, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestMultiSigIncorrectSignsTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	msigAddress := cState.Accounts.CreateMultisig([]uint32{1}, []types.Address{addr}, 1, accounts.CreateMultisigAddress(addr, 1))
	cState.Accounts.AddBalance(msigAddress, coin, helpers.BipToPip(big.NewInt(1000000)))

	txData := SendData{
		Coin:  types.GetBaseCoinID(),
		To:    types.Address{},
		Value: big.NewInt(1),
	}
	encodedData, _ := rlp.EncodeToBytes(txData)

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
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

	response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, &sync.Map{}, 0, false)

	if response.Code != code.IncorrectMultiSignature {
		t.Fatalf("Error code is not %d, got %d", code.IncorrectMultiSignature, response.Code)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestCustomCommissionCoinAndCustomGasCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := types.GetBaseCoinID()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	commissionCoin := createNonReserveCoin(cState)
	sendPrice := helpers.BipToPip(big.NewInt(1))
	{

		cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1)))

		poolBase := big.NewInt(100000)
		poolCustom := big.NewInt(10000)
		cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(poolBase))
		cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(poolBase))
		cState.Accounts.SubBalance(types.Address{}, commissionCoin, helpers.BipToPip(poolCustom))
		cState.Accounts.AddBalance(addr, commissionCoin, helpers.BipToPip(poolCustom))

		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.BipToPip(poolBase),
			Coin1:   commissionCoin,
			Volume1: helpers.BipToPip(poolCustom),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         1,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeCreateSwapPool,
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
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	price := commissionPrice
	price.Coin = commissionCoin
	price.Send = big.NewInt(0).Set(sendPrice)
	cState.Commission.SetNewCommissions(price.Encode())

	value := helpers.BipToPip(big.NewInt(10))
	cState.Accounts.AddBalance(addr, coin, value)

	cState.Accounts.SubBalance(types.Address{}, commissionCoin, sendPrice)
	cState.Accounts.AddBalance(addr, commissionCoin, sendPrice)

	to := types.Address([20]byte{1})

	data := SendData{
		Coin:  coin,
		To:    to,
		Value: value,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         2,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       commissionCoin,
		Type:          TypeSend,
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
		t.Fatalf("Response code is not 0. Error: %s, %s", response.Log, response.Info)
	}
	// for _, tag := range response.Tags {
	// t.Logf("%s: %s", tag.Key, tag.Value)
	// }

	targetBalance, _ := big.NewInt(0).SetString("0", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetBalance, balance)
	}

	commissionTargetBalance, _ := big.NewInt(0).SetString("0", 10)
	commissionCoinBalance := cState.Accounts.GetBalance(addr, commissionCoin)
	if commissionCoinBalance.Cmp(commissionTargetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), commissionTargetBalance, commissionCoinBalance)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("10000000000000000000", 10)
	testBalance := cState.Accounts.GetBalance(to, coin)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", to.String(), targetTestBalance, testBalance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestTwoTxFormOneSenderOneBlock(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.AddBalance(addr, 0, helpers.BipToPip(big.NewInt(1000000)))
	mempool := &sync.Map{}
	{

		txData := SendData{
			Coin:  types.GetBaseCoinID(),
			To:    types.Address{1},
			Value: big.NewInt(1),
		}
		encodedData, _ := rlp.EncodeToBytes(txData)

		tx := Transaction{
			Nonce:         1,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeSend,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		err := tx.Sign(privateKey)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		if response := NewExecutor(GetData).RunTx(state.NewCheckState(cState), txBytes, nil, 0, mempool, 0, false); response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}

		response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, mempool, 0, false)
		if response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}
	}
	{

		txData := SendData{
			Coin:  types.GetBaseCoinID(),
			To:    types.Address{1},
			Value: big.NewInt(1),
		}
		encodedData, _ := rlp.EncodeToBytes(txData)

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeSend,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		err := tx.Sign(privateKey)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		if response := NewExecutor(GetData).RunTx(state.NewCheckState(cState), txBytes, nil, 0, mempool, 0, false); response.Code != code.TxFromSenderAlreadyInMempool {
			t.Fatalf("Error code is not %d, got %d", code.TxFromSenderAlreadyInMempool, response.Code)
		}
	}
}
func TestTwoTxFormOneSenderAnyBlock(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.AddBalance(addr, 0, helpers.BipToPip(big.NewInt(1000000)))
	mempool := &sync.Map{}
	{

		txData := SendData{
			Coin:  types.GetBaseCoinID(),
			To:    types.Address{1},
			Value: big.NewInt(1),
		}
		encodedData, _ := rlp.EncodeToBytes(txData)

		tx := Transaction{
			Nonce:         1,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeSend,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		err := tx.Sign(privateKey)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		if response := NewExecutor(GetData).RunTx(state.NewCheckState(cState), txBytes, nil, 0, mempool, 0, false); response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}

		response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, mempool, 0, false)
		if response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}
	}
	mempool = &sync.Map{}
	{

		txData := SendData{
			Coin:  types.GetBaseCoinID(),
			To:    types.Address{1},
			Value: big.NewInt(1),
		}
		encodedData, _ := rlp.EncodeToBytes(txData)

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeSend,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		err := tx.Sign(privateKey)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		if response := NewExecutor(GetData).RunTx(state.NewCheckState(cState), txBytes, nil, 0, mempool, 0, false); response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}

		response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, mempool, 0, false)
		if response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}
	}
}
func TestTxOkAfterFailFormOneSenderInOneBlock(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	cState.Accounts.AddBalance(addr, 0, helpers.BipToPip(big.NewInt(1000000)))
	mempool := &sync.Map{}
	{

		txData := SendData{
			Coin:  types.GetBaseCoinID(),
			To:    types.Address{1},
			Value: big.NewInt(1),
		}
		encodedData, _ := rlp.EncodeToBytes(txData)

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeSend,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		err := tx.Sign(privateKey)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		if response := NewExecutor(GetData).RunTx(state.NewCheckState(cState), txBytes, nil, 0, mempool, 0, false); response.Code != code.WrongNonce {
			t.Fatalf("Error code is not %d, got %d", code.WrongNonce, response.Code)
		}

		response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, mempool, 0, false)
		if response.Code != code.WrongNonce {
			t.Fatalf("Error code is not %d, got %d", code.WrongNonce, response.Code)
		}
	}
	{

		txData := SendData{
			Coin:  types.GetBaseCoinID(),
			To:    types.Address{1},
			Value: big.NewInt(1),
		}
		encodedData, _ := rlp.EncodeToBytes(txData)

		tx := Transaction{
			Nonce:         1,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeSend,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		err := tx.Sign(privateKey)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		txBytes, err := rlp.EncodeToBytes(tx)
		if err != nil {
			t.Fatalf("Error %s", err.Error())
		}

		if response := NewExecutor(GetData).RunTx(state.NewCheckState(cState), txBytes, nil, 0, mempool, 0, false); response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}

		response := NewExecutor(GetData).RunTx(cState, txBytes, big.NewInt(0), 0, mempool, 0, false)
		if response.Code != code.OK {
			t.Fatalf("Error code is not %d, got %d", code.OK, response.Code)
		}
	}
}
