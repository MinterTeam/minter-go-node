package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
	"testing"
)

func getState() *state.StateDB {
	s, err := state.New(0, db.NewMemDB())

	if err != nil {
		panic(err)
	}

	createTestCoin(s)

	return s
}

func getTestCoinSymbol() types.CoinSymbol {
	var coin types.CoinSymbol
	copy(coin[:], []byte("TEST"))

	return coin
}

func createTestCoin(stateDB *state.StateDB) {
	volume := helpers.BipToPip(big.NewInt(100))
	reserve := helpers.BipToPip(big.NewInt(100))

	stateDB.CreateCoin(getTestCoinSymbol(), "TEST COIN", volume, 10, reserve, types.Address{})
}

func TestBuyCoinTx(t *testing.T) {
	cState := getState()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoin()

	cState.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	toBuy := helpers.BipToPip(big.NewInt(10))
	data := BuyCoinData{
		CoinToBuy:  getTestCoinSymbol(),
		ValueToBuy: toBuy,
		CoinToSell: coin,
	}

	encodedData, err := rlp.EncodeToBytes(data)

	if err != nil {
		t.Fatal(err)
	}

	buyCoinTx := Transaction{
		Nonce:         1,
		GasPrice:      big.NewInt(1),
		GasCoin:       coin,
		Type:          TypeBuyCoin,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	if err := buyCoinTx.Sign(privateKey); err != nil {
		t.Fatal(err)
	}

	encodedTx, err := rlp.EncodeToBytes(buyCoinTx)

	if err != nil {
		t.Fatal(err)
	}

	response := RunTx(cState, false, encodedTx, big.NewInt(0), 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999840525753990000000000", 10)
	balance := cState.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	testBalance := cState.GetBalance(addr, getTestCoinSymbol())
	if testBalance.Cmp(toBuy) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", getTestCoinSymbol(), toBuy, testBalance)
	}
}
