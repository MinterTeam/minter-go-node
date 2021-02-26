package transaction

import (
	"crypto/sha256"
	"math/big"
	"sync"
	"testing"

	c "github.com/MinterTeam/minter-go-node/coreV2/check"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"golang.org/x/crypto/sha3"
)

func TestRedeemCheckTx(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)

	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)

	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
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

	balance := cState.Accounts.GetBalance(receiverAddr, coin)
	if balance.Cmp(checkValue) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, checkValue, balance)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToDecodeError(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)

	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)

	if err != nil {
		t.Fatal(err)
	}

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		Proof: proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	response := data.basicCheck(&tx, state.NewCheckState(cState))
	if response.Code != code.DecodeError {
		t.Fatalf("Response code is not %d. Error %s", code.DecodeError, response.Log)
	}

	data.RawCheck = []byte{0}
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx.Data = encodedData
	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	txResponse := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if txResponse.Code != code.DecodeError {
		t.Fatalf("Response code is not %d. Error %s", code.DecodeError, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToHighGasPrice(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      2,
		ChainID:       types.CurrentChainID,
		GasCoin:       coin,
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.TooHighGasPrice {
		t.Fatalf("Response code is not %d. Error %s", code.TooHighGasPrice, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToWrongChainID(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.ChainTestnet,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.WrongChainID {
		t.Fatalf("Response code is not %d. Error %s", code.WrongChainID, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToNonceLength(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.TooLongNonce {
		t.Fatalf("Response code is not %d. Error %s", code.TooLongNonce, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToCheckData(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     5,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	check.Coin = coin
	check.GasCoin = 5
	lock, err = crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)
	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ = rlp.EncodeToBytes(check)
	data = RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
	}

	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx.Data = encodedData
	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CoinNotExists {
		t.Fatalf("Response code is not %d. Error %s", code.CoinNotExists, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	check.GasCoin = coin
	lock, err = crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)
	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ = rlp.EncodeToBytes(check)
	data.RawCheck = rawCheck
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	customCoin := createTestCoin(cState)
	tx.GasCoin = customCoin
	tx.Data = encodedData
	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.WrongGasCoin {
		t.Fatalf("Response code is not %d. Error %s", code.WrongGasCoin, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	check.DueBlock = 1
	lock, err = crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)
	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ = rlp.EncodeToBytes(check)
	data.RawCheck = rawCheck
	encodedData, err = rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx.GasCoin = coin
	tx.Data = encodedData
	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response = RunTx(cState, encodedTx, big.NewInt(0), 100, &sync.Map{}, 0)
	if response.Code != code.CheckExpired {
		t.Fatalf("Response code is not %d. Error %s", code.CheckExpired, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToUsed(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	cState.Checks.UseCheck(&check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CheckUsed {
		t.Fatalf("Response code is not %d. Error %s", code.CheckUsed, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToInsufficientFunds(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := types.GetBaseCoinID()

	senderPrivateKey, _ := crypto.GenerateKey()

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToCoinReserveUnderflow(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := createTestCoin(cState)
	cState.Coins.SubReserve(coin, helpers.BipToPip(big.NewInt(90000)))

	senderPrivateKey, _ := crypto.GenerateKey()

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  coin,
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.CommissionCoinNotSufficient {
		t.Fatalf("Response code is not %d. Error %s", code.CommissionCoinNotSufficient, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToInsufficientFundsForCheckCoin(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := createTestCoin(cState)

	senderPrivateKey, _ := crypto.GenerateKey()

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestRedeemCheckTxToInsufficientFundsForCheckGasCoin(t *testing.T) {
	t.Parallel()
	cState := getState()
	coin := createTestCoin(cState)

	senderPrivateKey, senderAddr := getAccount()

	receiverPrivateKey, _ := crypto.GenerateKey()
	receiverAddr := crypto.PubkeyToAddress(receiverPrivateKey.PublicKey)

	passphrase := "password"
	passphraseHash := sha256.Sum256([]byte(passphrase))
	passphrasePk, err := crypto.ToECDSA(passphraseHash[:])

	cState.Coins.AddVolume(coin, helpers.BipToPip(big.NewInt(100)))
	cState.Accounts.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(100)))

	if err != nil {
		t.Fatal(err)
	}

	checkValue := helpers.BipToPip(big.NewInt(10))

	check := c.Check{
		Nonce:    []byte{1, 2, 3},
		ChainID:  types.CurrentChainID,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
		GasCoin:  types.GetBaseCoinID(),
	}

	lock, err := crypto.Sign(check.HashWithoutLock().Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	check.Lock = big.NewInt(0).SetBytes(lock)

	err = check.Sign(senderPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	rawCheck, _ := rlp.EncodeToBytes(check)

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		receiverAddr,
	})
	hw.Sum(senderAddressHash[:0])

	sig, err := crypto.Sign(senderAddressHash.Bytes(), passphrasePk)
	if err != nil {
		t.Fatal(err)
	}

	proof := [65]byte{}
	copy(proof[:], sig)

	data := RedeemCheckData{
		RawCheck: rawCheck,
		Proof:    proof,
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
		Type:          TypeRedeemCheck,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := tx.Sign(receiverPrivateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != code.InsufficientFunds {
		t.Fatalf("Response code is not %d. Error %s", code.InsufficientFunds, response.Log)
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
