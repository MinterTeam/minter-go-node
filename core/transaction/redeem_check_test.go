package transaction

import (
	"crypto/sha256"
	c "github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/crypto/sha3"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sync"
	"testing"
)

func TestRedeemCheckTx(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}

func TestRedeemCheckTxToDecodeError(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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

	response := data.BasicCheck(&tx, state.NewCheckState(cState))
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
}

func TestRedeemCheckTxToHighGasPrice(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}

func TestRedeemCheckTxToWrongChainID(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}

func TestRedeemCheckTxToNonceLength(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}

func TestRedeemCheckTxToCheckData(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}

func TestRedeemCheckTxToUsed(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}

func TestRedeemCheckTxToInsufficientFunds(t *testing.T) {
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
	hw := sha3.NewKeccak256()
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
}
