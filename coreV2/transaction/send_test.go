package transaction

import (
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestSendTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
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
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999989990000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetBalance, balance)
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

func TestSendMultisigTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	coin := types.GetBaseCoinID()

	msig := cState.Accounts.CreateMultisig([]uint32{1, 1}, []types.Address{addr1, addr2}, 1, accounts.CreateMultisigAddress(addr1, 1))

	cState.Accounts.AddBalance(msig, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
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
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeSend,
		Data:          encodedData,
		SignatureType: SigTypeMulti,
	}

	if err := tx.Sign(privateKey1); err != nil {
		t.Fatal(err)
	}

	tx.SetMultisigAddress(msig)

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999989990000000000000000", 10)
	balance := cState.Accounts.GetBalance(msig, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", msig.String(), targetBalance, balance)
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

func TestSendFailedMultisigTx(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(privateKey1.PublicKey)

	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	coin := types.GetBaseCoinID()

	msig := cState.Accounts.CreateMultisig([]uint32{1, 3}, []types.Address{addr1, addr2}, 3, accounts.CreateMultisigAddress(addr1, 1))

	cState.Accounts.AddBalance(msig, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
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
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeSend,
		Data:          encodedData,
		SignatureType: SigTypeMulti,
	}

	if err := tx.Sign(privateKey1); err != nil {
		t.Fatal(err)
	}

	tx.SetMultisigAddress(msig)

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.NotEnoughMultisigVotes {
		t.Fatalf("Response code is not %d. Gor: %d", code.NotEnoughMultisigVotes, response.Code)
	}

	targetBalance, _ := big.NewInt(0).SetString("1000000000000000000000000", 10)
	balance := cState.Accounts.GetBalance(msig, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", msig.String(), targetBalance, balance)
	}

	targetTestBalance, _ := big.NewInt(0).SetString("0", 10)
	testBalance := cState.Accounts.GetBalance(to, coin)
	if testBalance.Cmp(targetTestBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", to.String(), targetTestBalance, testBalance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSendWithNotExistedCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	coin := types.CoinID(5)

	value := helpers.BipToPip(big.NewInt(10))
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
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestSendTxWithCustomCoin(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)

	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
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
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error: %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999989988999999504999969", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), targetBalance, balance)
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

func TestSendTxToGasCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := createTestCoin(cState)

	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))
	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
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
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error: %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
