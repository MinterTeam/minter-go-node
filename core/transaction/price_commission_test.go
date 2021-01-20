package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func TestPriceCommissionTx(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coin1 := createNonReserveCoin(cState)
	cState.Accounts.SubBalance(types.Address{}, coin1, big.NewInt(1e18))

	cState.Swap.PairMint(types.Address{}, types.GetBaseCoinID(), coin1, big.NewInt(1e18), big.NewInt(1e18))
	// cState.Accounts.SubBalance(addr, coin1, big.NewInt(1e18))
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), big.NewInt(1e18))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	{
		data := []interface{}{
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			coin1,
			pubkey,
			uint64(100500),
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
			Type:          TypePriceCommission,
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

		response := RunTx(cState, encodedTx, &commission.Price{}, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != 0 {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	{
		data := []interface{}{
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			coin1,
			pubkey,
			uint64(100500),
		}
		encodedData, err := rlp.EncodeToBytes(data)
		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypePriceCommission,
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

		response := RunTx(cState, encodedTx, &commission.Price{}, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != code.VoiceAlreadyExists {
			t.Fatalf("Response code is not %d. Error: %s", code.VoiceAlreadyExists, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestPriceCommissionDeleteTx(t *testing.T) {
	t.Parallel()
	cState := getState()
	privateKey, addr := getAccount()
	coin1 := createNonReserveCoin(cState)
	cState.Accounts.SubBalance(types.Address{}, coin1, big.NewInt(1e18))

	cState.Swap.PairMint(types.Address{}, types.GetBaseCoinID(), coin1, big.NewInt(1e18), big.NewInt(1e18))
	cState.Accounts.AddBalance(addr, types.GetBaseCoinID(), big.NewInt(1e18))

	pubkey := [32]byte{}
	rand.Read(pubkey[:])

	cState.Candidates.Create(addr, addr, addr, pubkey, 10, 0)
	cState.Validators.Create(pubkey, helpers.BipToPip(big.NewInt(1)))
	{
		data := []interface{}{
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			coin1,
			pubkey,
			uint64(100500),
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
			Type:          TypePriceCommission,
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

		response := RunTx(cState, encodedTx, &commission.Price{}, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != 0 {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	cState.Commission.Delete(100500)
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
	{
		data := []interface{}{
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			big.NewInt(1e18),
			coin1,
			pubkey,
			uint64(100500),
		}
		encodedData, err := rlp.EncodeToBytes(data)
		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypePriceCommission,
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

		response := RunTx(cState, encodedTx, &commission.Price{}, big.NewInt(0), 0, &sync.Map{}, 0)
		if response.Code != code.OK {
			t.Fatalf("Response code is not 0. Error: %s", response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
