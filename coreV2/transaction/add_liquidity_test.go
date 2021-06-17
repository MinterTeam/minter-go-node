package transaction

import (
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/state"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func createNonReserveCoin(stateDB *state.State) types.CoinID {
	volume := helpers.BipToPip(big.NewInt(100000))

	id := stateDB.App.GetNextCoinID()
	stateDB.Coins.CreateToken(id, types.StrToCoinSymbol("TOKEN"+id.String()), "TOKEN"+id.String(), true, true, volume, big.NewInt(0).Mul(volume, big.NewInt(10)), nil)
	stateDB.App.SetCoinsCount(id.Uint32())
	stateDB.Accounts.AddBalance(types.Address{}, id, volume)

	return id
}

func TestAddExchangeLiquidityTx_Bug4415829(t *testing.T) {
	t.Parallel()
	cState := getState()
	commissionPrice := commissionPrice
	commissionPrice.AddLiquidity = helpers.StringToBigInt("5000000000000000000")
	cState.Commission.SetNewCommissions(commissionPrice.Encode())
	coin := types.GetBaseCoinID()
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.StringToBigInt("4969149711623110"))
	cState.Accounts.AddBalance(addr, coin1, helpers.StringToBigInt("4969149711623110"))

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.StringToBigInt("1548455047339648479671"),
			Coin1:   coin1,
			Volume1: helpers.StringToBigInt("4694569815457508"),
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

		response := NewExecutor(GetDataV230).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != 0 {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := AddLiquidityData{
			Coin0:          coin,
			Volume0:        helpers.StringToBigInt("85867881698893382390"),
			Coin1:          coin1,
			MaximumVolume1: helpers.StringToBigInt("273300000000000"),
		}

		encodedData, err := rlp.EncodeToBytes(data)

		if err != nil {
			t.Fatal(err)
		}

		tx := Transaction{
			Nonce:         2,
			GasPrice:      1,
			ChainID:       types.CurrentChainID,
			GasCoin:       coin1,
			Type:          TypeAddLiquidity,
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

		{
			response := NewExecutor(GetDataV240).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

			if response.Code != 107 {
				t.Fatalf("Response code %d is not 107. Error: %s", response.Code, response.Log)
			}
		}

		{
			response := NewExecutor(GetDataV230).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

			if response.Code != 0 {
				t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
			}

			if cState.Accounts.GetBalance(addr, coin1).Sign() == 1 {
				t.Error("is not bug")
			}

			if err := checkState(cState); err == nil {
				t.Error("error")
			}
		}
	}

}

func TestAddExchangeLiquidityTx_initialLiquidity(t *testing.T) {
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

func TestAddExchangeLiquidityTx_initialLiquidity_1(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Checker.AddCoin(types.BasecoinID, helpers.StringToBigInt("-1099999999000000000000000"))
	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	data := CreateSwapPoolData{
		Coin0:   coin,
		Volume0: helpers.BipToPip(big.NewInt(9)),
		Coin1:   coin1,
		Volume1: helpers.BipToPip(big.NewInt(11)),
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

func TestAddExchangeLiquidityTx_addLiquidity(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	cState.Checker.AddCoin(types.BasecoinID, helpers.StringToBigInt("-1099999999000000000000000"))
	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr2, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.AddBalance(addr2, coin, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.AddBalance(addr2, coin1, helpers.BipToPip(big.NewInt(50000)))

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: big.NewInt(10000),
			Coin1:   coin1,
			Volume1: big.NewInt(10000),
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
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
	{
		data := AddLiquidityData{
			Coin0:          coin,
			Volume0:        helpers.BipToPip(big.NewInt(10)),
			Coin1:          coin1,
			MaximumVolume1: helpers.BipToPip(big.NewInt(10)),
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
			Type:          TypeAddLiquidity,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		if err := tx.Sign(privateKey2); err != nil {
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
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestAddExchangeLiquidityTx_addLiquidity_1(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	cState.Checker.AddCoin(types.BasecoinID, helpers.StringToBigInt("-1099999999000000000000000"))
	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr2, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.AddBalance(addr2, coin, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.AddBalance(addr2, coin1, helpers.BipToPip(big.NewInt(50000)))

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
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
	{
		data := AddLiquidityData{
			Coin0:          coin,
			Volume0:        big.NewInt(10000),
			Coin1:          coin1,
			MaximumVolume1: big.NewInt(10000),
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
			Type:          TypeAddLiquidity,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		if err := tx.Sign(privateKey2); err != nil {
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
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}

func TestAddExchangeLiquidityTx_addLiquidity_2(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin := createTestCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	privateKey2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(privateKey2.PublicKey)

	cState.Checker.AddCoin(types.BasecoinID, helpers.StringToBigInt("-1099999999000000000000000"))
	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))
	cState.Accounts.AddBalance(addr2, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.AddBalance(addr2, coin, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(50000)))
	cState.Accounts.AddBalance(addr2, coin1, helpers.BipToPip(big.NewInt(50000)))

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.BipToPip(big.NewInt(9)),
			Coin1:   coin1,
			Volume1: helpers.BipToPip(big.NewInt(11)),
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
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
	{
		data := AddLiquidityData{
			Coin0:          coin,
			Volume0:        big.NewInt(9000),
			Coin1:          coin1,
			MaximumVolume1: big.NewInt(11000),
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
			Type:          TypeAddLiquidity,
			Data:          encodedData,
			SignatureType: SigTypeSingle,
		}

		if err := tx.Sign(privateKey2); err != nil {
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
	}
	if err := checkState(cState); err != nil {
		t.Error(err)
	}
}
