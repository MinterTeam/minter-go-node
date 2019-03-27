package transaction

import (
	"crypto/sha256"
	c "github.com/MinterTeam/minter-go-node/core/check"
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
	coin := types.GetBaseCoin()

	senderPrivateKey, _ := crypto.GenerateKey()
	senderAddr := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	cState.AddBalance(senderAddr, coin, helpers.BipToPip(big.NewInt(1000000)))

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
		Nonce:    0,
		DueBlock: 1,
		Coin:     coin,
		Value:    checkValue,
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
		GasPrice:      big.NewInt(1),
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

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0, sync.Map{}, big.NewInt(0))

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	balance := cState.GetBalance(receiverAddr, coin)
	if balance.Cmp(checkValue) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, checkValue, balance)
	}
}
