package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"sync"
	"testing"
)

func TestMintData_aaa(t *testing.T) {
	t.Parallel()
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toCreate := types.StrToCoinSymbol("TOKEN1")
	amount := helpers.BipToPip(big.NewInt(100))
	name := "My Test Coin"
	{
		data := CreateTokenData{
			Name:          name,
			Symbol:        toCreate,
			InitialAmount: amount,
			MaxSupply:     big.NewInt(0).Mul(amount, big.NewInt(10)),
			Mintable:      true,
			Burnable:      false,
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
			Type:          TypeCreateToken,
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
			t.Fatalf("Response code is not 0. Error %s", response.Log)
		}

		_, _, err = cState.Tree().Commit(cState.Coins)
		if err != nil {
			t.Fatalf("Commit coins failed. Error %s", err)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}

		targetBalance, _ := big.NewInt(0).SetString("999000000000000000000000", 10)
		balance := cState.Accounts.GetBalance(addr, coin)
		if balance.Cmp(targetBalance) != 0 {
			t.Errorf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
		}

		stateCoin := cState.Coins.GetCoinBySymbol(toCreate, 0)

		if stateCoin == nil {
			t.Fatalf("Coin %s not found in state", toCreate)
		}

		if stateCoin.Volume().Cmp(amount) != 0 {
			t.Fatalf("Volume in state is not correct. Expected %s, got %s", amount, stateCoin.MaxSupply())
		}

		if stateCoin.Name() != name {
			t.Fatalf("Name in state is not correct. Expected %s, got %s", name, stateCoin.Name())
		}

		if stateCoin.Version() != 0 {
			t.Fatalf("Version in state is not correct. Expected %d, got %d", 0, stateCoin.Version())
		}

		symbolInfo := cState.Coins.GetSymbolInfo(toCreate)
		if symbolInfo == nil {
			t.Fatalf("Symbol %s info not found in state", toCreate)
		}

		if *symbolInfo.OwnerAddress() != addr {
			t.Fatalf("Target owner address is not correct. Expected %s, got %s", addr.String(), symbolInfo.OwnerAddress().String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	{
		stateCoin := cState.Coins.GetCoinBySymbol(toCreate, 0)

		if stateCoin == nil {
			t.Fatalf("Coin %s not found in state", toCreate)
		}

		if stateCoin.Volume().Cmp(amount) != 0 {
			t.Fatalf("Volume in state is not correct. Expected %s, got %s", amount, stateCoin.MaxSupply())
		}

		subVolume := big.NewInt(1e18)
		data := MintTokenData{
			Coin:  stateCoin.ID(),
			Value: big.NewInt(0).Set(subVolume),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       coin,
			Type:          TypeMintToken,
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
			t.Fatalf("Response code is not 0. Error %s", response.Log)
		}

		_, _, err = cState.Tree().Commit(cState.Coins)
		if err != nil {
			t.Fatalf("Commit coins failed. Error %s", err)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}

		stateCoin = cState.Coins.GetCoinBySymbol(toCreate, 0)

		if stateCoin == nil {
			t.Fatalf("Coin %s not found in state", toCreate)
		}

		amount.Add(amount, subVolume)

		if stateCoin.Volume().Cmp(amount) != 0 {
			t.Errorf("Volume in state is not correct. Expected %s, got %s", amount, stateCoin.MaxSupply())
		}

		balance := cState.Accounts.GetBalance(addr, stateCoin.ID())
		if balance.Cmp(amount) != 0 {
			t.Errorf("Target %s balance is not correct. Expected %s, got %s", stateCoin.ID(), amount, balance)
		}
	}
}
