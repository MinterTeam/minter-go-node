package transaction

import (
	"github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"math/big"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func TestAddOrderSwapPoolData_RO(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	cState := getState()

	coin1 := createNonReserveCoin(cState)
	coin := types.GetBaseCoinID()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

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

	commissionPrice := commissionPrice
	commissionPrice.Coin = coin1
	cState.Commission.SetNewCommissions(commissionPrice.Encode())

	{
		data := AddLimitOrderData{
			CoinToSell:  coin,
			ValueToSell: helpers.BipToPip(big.NewInt(25)),
			CoinToBuy:   coin1,
			ValueToBuy:  helpers.BipToPip(big.NewInt(25)),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.WrongOrderPrice {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
func TestAddOrderSwapPoolData_RO3(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	cState := getState()
	coin1 := createNonReserveCoin(cState)

	coin := types.GetBaseCoinID()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000000)))

	//cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin1,
			Volume0: helpers.StringToBigInt("29122939232932932139390"),
			Coin1:   coin,
			Volume1: helpers.StringToBigInt("2912293923293293213939"),
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

		response := NewExecutorV250(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != 0 {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	commissionPrice := commissionPrice
	commissionPrice.Coin = coin1
	cState.Commission.SetNewCommissions(commissionPrice.Encode())

	{
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.StringToBigInt("2912293923293293213939"),
			CoinToBuy:   coin,
			ValueToBuy:  big.NewInt(0).Sub(helpers.BipToPip(helpers.StringToBigInt("29122939232932932139390")), big.NewInt(1)),
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
			Type:          TypeAddLimitOrder,
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

		response := NewExecutorV250(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != code.WrongOrderPrice {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		for _, tag := range response.Tags {
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	{
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: big.NewInt(0).Add(helpers.StringToBigInt("29122939232932932139390"), big.NewInt(1)),
			CoinToBuy:   coin,
			ValueToBuy:  helpers.StringToBigInt("2912293923293293213939"),
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
			Type:          TypeAddLimitOrder,
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

		response := NewExecutorV250(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != 0 {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		for _, tag := range response.Tags {
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
func TestAddOrderSwapPoolData_RO2(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	cState := getState()
	coin1 := createNonReserveCoin(cState)

	coin := types.GetBaseCoinID()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000000)))

	//cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.StringToBigInt("2912293923293293213939000"),
			Coin1:   coin1,
			Volume1: helpers.StringToBigInt("2912293923293293213939"),
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

		response := NewExecutorV250(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != 0 {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	commissionPrice := commissionPrice
	commissionPrice.Coin = coin1
	cState.Commission.SetNewCommissions(commissionPrice.Encode())

	{
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.StringToBigInt("2912293923293293213939"),
			CoinToBuy:   coin,
			ValueToBuy:  big.NewInt(0).Sub(helpers.BipToPip(helpers.StringToBigInt("2912293923293293213939000")), big.NewInt(1)),
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
			Type:          TypeAddLimitOrder,
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

		response := NewExecutorV250(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != code.WrongOrderPrice {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		for _, tag := range response.Tags {
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	{
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.StringToBigInt("2912293923293293213939"),
			CoinToBuy:   coin,
			ValueToBuy:  big.NewInt(0).Sub(helpers.StringToBigInt("2912293923293293213939000"), big.NewInt(1)),
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
			Type:          TypeAddLimitOrder,
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

		response := NewExecutorV250(GetData).RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0, false)

		if response.Code != 0 {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		for _, tag := range response.Tags {
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
func TestAddOrderSwapPoolData_RO1(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	cState := getState()

	coin1 := createNonReserveCoin(cState)
	coin := types.GetBaseCoinID()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000000)))

	//cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin,
			Volume0: helpers.StringToBigInt("2912293923293293213939000"),
			Coin1:   coin1,
			Volume1: helpers.StringToBigInt("2912293923293293213939"),
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.StringToBigInt("2912293923293293213939"),
			CoinToBuy:   coin,
			ValueToBuy:  big.NewInt(0).Sub(helpers.BipToPip(helpers.StringToBigInt("2912293923293293213939000")), big.NewInt(1)),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.WrongOrderPrice {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		for _, tag := range response.Tags {
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	{
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.StringToBigInt("2912293923293293213939"),
			CoinToBuy:   coin,
			ValueToBuy:  big.NewInt(0).Sub(helpers.StringToBigInt("2912293923293293213939000"), big.NewInt(1)),
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
			Type:          TypeAddLimitOrder,
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

		for _, tag := range response.Tags {
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
func TestAddOrderSwapPoolData_WrongPrice(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	cState := getState()

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(25)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.WrongOrderPrice {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
	{
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(35)),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.WrongOrderPrice {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestAddOrderSwapPoolData_ExpireLittleOrder_sell(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	e := &events.MockEvents{}
	cState := getState(e)

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: big.NewInt(1e10),
			CoinToBuy:   coin0,
			ValueToBuy:  big.NewInt(1e10),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.OK {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	{
		data := SellSwapPoolDataV260{
			Coins:             []types.CoinID{coin0, coin1},
			ValueToSell:       big.NewInt(10010000000 - 1),
			MinimumValueToBuy: big.NewInt(1),
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
			Type:          TypeSellSwapPool,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	for _, event := range e.LoadEvents(0) {
		t.Logf("%#v", event.(*events.OrderExpiredEvent))
	}
}

func TestAddOrderSwapPoolData_ExpireLittleOrder_sell_MinimumValueToBuyReached(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	e := &events.MockEvents{}
	cState := getState(e)

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: big.NewInt(1e10),
			CoinToBuy:   coin0,
			ValueToBuy:  big.NewInt(1e10),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.OK {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	{
		data := SellSwapPoolDataV260{
			Coins:             []types.CoinID{coin0, coin1},
			ValueToSell:       big.NewInt(1),
			MinimumValueToBuy: big.NewInt(0),
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
			Type:          TypeSellSwapPool,
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

		if response.Code != code.MinimumValueToBuyReached {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

}
func TestAddOrderSwapPoolData_ExpireLittleOrder_buy(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	e := &events.MockEvents{}
	cState := getState(e)

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: big.NewInt(1e10),
			CoinToBuy:   coin0,
			ValueToBuy:  big.NewInt(1e10),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.OK {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	{
		data := BuySwapPoolDataV260{
			Coins:              []types.CoinID{coin0, coin1},
			MaximumValueToSell: big.NewInt(1e18),
			ValueToBuy:         big.NewInt(9990000000 - 1),
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	for _, event := range e.LoadEvents(0) {
		t.Logf("%#v", event.(*events.OrderExpiredEvent))
	}
}
func TestAddOrderSwapPoolData_InsufficientLiquidity_buy(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	e := &events.MockEvents{}
	cState := getState(e)

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
			Volume0: helpers.StringToBigInt("1650430261925377833"),
			Coin1:   coin1,
			Volume1: helpers.StringToBigInt("831046393127060170"),
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.StringToBigInt("381545748998466867"),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.StringToBigInt("760000000000000000"),
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
			Type:          TypeAddLimitOrder,
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

		if response.Code != code.OK {
			t.Fatalf("Response code %d is not 0. Error: %s", response.Code, response.Log)
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	{
		data := BuySwapPoolDataV260{
			Coins:              []types.CoinID{coin0, coin1},
			MaximumValueToSell: coins.MaxCoinSupply(),
			ValueToBuy:         big.NewInt(15e17),
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

		if response.Code != code.InsufficientLiquidity {
			t.Fatalf("Response code %d is not %d. Error: %s", response.Code, code.InsufficientLiquidity, response.Log)
		}

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}

	for _, event := range e.LoadEvents(0) {
		t.Logf("%#v", event.(*events.OrderExpiredEvent))
	}
}

func TestAddOrderSwapPoolData_Buy_01_moreOrder(t *testing.T) {
	// t.SkipNow()
	t.Parallel()
	cState := getState()

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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
		i := int64(23)
		v, _ := cState.Swapper().GetSwapper(coin0, coin1).CalculateBuyForSellWithOrders(helpers.BipToPip(big.NewInt(i)))
		data := BuySwapPoolDataV260{
			Coins:              []types.CoinID{coin0, coin1},
			MaximumValueToSell: helpers.BipToPip(big.NewInt(i)),
			ValueToBuy:         v,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
func TestAddOrderSwapPoolData_Buy_01_partOrder(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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
		i := int64(22)
		v, _ := cState.Swapper().GetSwapper(coin0, coin1).CalculateBuyForSellWithOrders(helpers.BipToPip(big.NewInt(i)))
		data := BuySwapPoolDataV260{
			Coins:              []types.CoinID{coin0, coin1},
			MaximumValueToSell: helpers.StringToBigIntOrNil("22000000000000000912"),
			ValueToBuy:         v,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
func TestAddOrderSwapPoolData_Sell_01_partOrder(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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
		i := int64(22)
		data := SellSwapPoolDataV260{
			Coins:             []types.CoinID{coin0, coin1},
			ValueToSell:       helpers.BipToPip(big.NewInt(i)),
			MinimumValueToBuy: cState.Swapper().GetSwapper(coin0, coin1).CalculateBuyForSell(helpers.BipToPip(big.NewInt(i))),
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
			Type:          TypeSellSwapPool,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestAddOrderSwapPoolData_Sell_10_partOrder(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin1 := createNonReserveCoin(cState)
	coin0 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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
		i := int64(22)
		data := SellSwapPoolDataV260{
			Coins:             []types.CoinID{coin0, coin1},
			ValueToSell:       helpers.BipToPip(big.NewInt(i)),
			MinimumValueToBuy: cState.Swapper().GetSwapper(coin0, coin1).CalculateBuyForSell(helpers.BipToPip(big.NewInt(i))),
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
			Type:          TypeSellSwapPool,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestAddOrderSwapPoolData_Sell_01_fullOrder(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin0 := createNonReserveCoin(cState)
	coin1 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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
		i := int64(23)
		data := SellSwapPoolDataV260{
			Coins:             []types.CoinID{coin0, coin1},
			ValueToSell:       helpers.BipToPip(big.NewInt(i)),
			MinimumValueToBuy: cState.Swapper().GetSwapper(coin0, coin1).CalculateBuyForSell(helpers.BipToPip(big.NewInt(i))),
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
			Type:          TypeSellSwapPool,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}

func TestAddOrderSwapPoolData_Sell_10_fullOrder(t *testing.T) {
	t.Parallel()
	cState := getState()

	coin1 := createNonReserveCoin(cState)
	coin0 := createNonReserveCoin(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	cState.Accounts.AddBalance(addr, types.BasecoinID, helpers.BipToPip(big.NewInt(1000000)))

	cState.Accounts.SubBalance(types.Address{}, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin0, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.SubBalance(types.Address{}, coin1, helpers.BipToPip(big.NewInt(100000)))
	cState.Accounts.AddBalance(addr, coin1, helpers.BipToPip(big.NewInt(100000)))

	if err := checkState(cState); err != nil {
		t.Error(err)
	}

	{
		data := CreateSwapPoolData{
			Coin0:   coin0,
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
		data := AddLimitOrderData{
			CoinToSell:  coin1,
			ValueToSell: helpers.BipToPip(big.NewInt(5)),
			CoinToBuy:   coin0,
			ValueToBuy:  helpers.BipToPip(big.NewInt(15)),
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
			Type:          TypeAddLimitOrder,
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
		i := int64(23)
		data := SellSwapPoolDataV260{
			Coins:             []types.CoinID{coin0, coin1},
			ValueToSell:       helpers.BipToPip(big.NewInt(i)),
			MinimumValueToBuy: cState.Swapper().GetSwapper(coin0, coin1).CalculateBuyForSell(helpers.BipToPip(big.NewInt(i))),
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
			Type:          TypeSellSwapPool,
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

		for _, tag := range response.Tags {
			if string(tag.Key) != "tx.pools" {
				continue
			}
			t.Log(tag.String())
		}

		if err := checkState(cState); err != nil {
			t.Error(err)
		}
	}
}
