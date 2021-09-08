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

func TestCalculateCommissionTODO(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coinSend := types.GetBaseCoinID()
	coinCommission := createNonReserveCoin(cState)

	cState.Accounts.AddBalance(addr, coinSend, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr, coinCommission, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(10))
	addressTo := types.Address([20]byte{1})

	data := SendData{
		Coin:  coinSend,
		To:    addressTo,
		Value: value,
	}

	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      2,
		ChainID:       types.CurrentChainID,
		GasCoin:       coinCommission,
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

	response := NewExecutorV250(GetDataV250).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)
	if response.Code == 0 {
		t.Fatal("Response code is 0, want error")
	}

	if balance := cState.Accounts.GetBalance(addr, coinSend); balance.Cmp(helpers.BipToPip(big.NewInt(1000000))) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), helpers.BipToPip(big.NewInt(1000000)), balance)
	}

	if balance := cState.Accounts.GetBalance(addr, coinCommission); balance.Cmp(helpers.BipToPip(big.NewInt(1000000))) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", addr.String(), helpers.BipToPip(big.NewInt(1000000)), balance)
	}
}
