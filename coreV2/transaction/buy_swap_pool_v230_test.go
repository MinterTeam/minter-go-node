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

func TestBuySwapPoolTx_0(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.BipToPip(big.NewInt(100)),
			Coin1:   coin1,
			Volume1: helpers.BipToPip(big.NewInt(1000)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1},
			MaximumValueToSell: big.NewInt(10),
			ValueToBuy:         big.NewInt(99),
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
			Type:          TypeBuySwapPool,
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
}

func TestBuySwapPoolTx_1(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.BipToPip(big.NewInt(10)),
			Coin1:   coin1,
			Volume1: helpers.BipToPip(big.NewInt(10)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin1, coin},
			MaximumValueToSell: big.NewInt(10),
			ValueToBuy:         big.NewInt(9),
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
			Type:          TypeBuySwapPool,
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1},
			MaximumValueToSell: big.NewInt(10),
			ValueToBuy:         big.NewInt(9),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         3,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeBuySwapPool,
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
}

func TestBuySwapPoolTx_2(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: new(big.Int).Add(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)), big.NewInt(0)),
			Coin1:   coin1,
			Volume1: new(big.Int).Add(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)), big.NewInt(0)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1},
			MaximumValueToSell: new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			ValueToBuy:         big.NewInt(996006981039903216),
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
			Type:          TypeBuySwapPool,
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
}

func TestBuySwapPoolTx_3(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: new(big.Int).Add(new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18)), big.NewInt(0)),
			Coin1:   coin1,
			Volume1: new(big.Int).Add(new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)), big.NewInt(0)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1},
			MaximumValueToSell: new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			ValueToBuy:         big.NewInt(1662497915624478906),
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
			Type:          TypeBuySwapPool,
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
}

func TestBuySwapPoolTx_4(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: new(big.Int).Add(new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)), big.NewInt(0)),
			Coin1:   coin1,
			Volume1: new(big.Int).Add(new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18)), big.NewInt(0)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1},
			MaximumValueToSell: new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			ValueToBuy:         big.NewInt(453305446940074565),
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
			Type:          TypeBuySwapPool,
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
}

func TestBuySwapPoolTx_RouteYes(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.BipToPip(big.NewInt(100)),
			Coin1:   coin1,
			Volume1: helpers.BipToPip(big.NewInt(1000)),
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
	coin2 := createNonReserveCoin(cState)
	cState.Accounts.SubBalance(types.Address{}, coin2, helpers.BipToPip(big.NewInt(1000)))
	cState.Accounts.AddBalance(addr, coin2, helpers.BipToPip(big.NewInt(1000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin1,
			Volume0: helpers.BipToPip(big.NewInt(1000)),
			Coin1:   coin2,
			Volume1: helpers.BipToPip(big.NewInt(1000)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1, coin2},
			MaximumValueToSell: big.NewInt(10),
			ValueToBuy:         big.NewInt(98),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         3,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeBuySwapPool,
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
}

func TestBuySwapPoolTx_Route(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.BipToPip(big.NewInt(100)),
			Coin1:   coin1,
			Volume1: helpers.BipToPip(big.NewInt(1000)),
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin, coin1},
			MaximumValueToSell: big.NewInt(10),
			ValueToBuy:         big.NewInt(99),
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
			Type:          TypeBuySwapPool,
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

	coin2 := createNonReserveCoin(cState)
	cState.Accounts.SubBalance(types.Address{}, coin2, helpers.BipToPip(big.NewInt(1000)))
	cState.Accounts.AddBalance(addr, coin2, helpers.BipToPip(big.NewInt(1000)))
	{
		data := CreateSwapPoolData{
			Coin0:   coin1,
			Volume0: helpers.BipToPip(big.NewInt(1000)),
			Coin1:   coin2,
			Volume1: helpers.BipToPip(big.NewInt(1000)),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         3,
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
	{
		data := BuySwapPoolDataV230{
			Coins:              []types.CoinID{coin1, coin2},
			MaximumValueToSell: big.NewInt(99),
			ValueToBuy:         big.NewInt(98),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         4,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       types.GetBaseCoinID(),
			Type:          TypeBuySwapPool,
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
}
