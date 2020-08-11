package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"

	"math/big"
	"math/rand"
	"sync"
	"testing"
)

func createTestCandidate(stateDB *state.State) types.Pubkey {
	address := types.Address{}
	pubkey := types.Pubkey{}
	rand.Read(pubkey[:])

	stateDB.Candidates.Create(address, address, address, pubkey, 10)

	return pubkey
}

func TestDelegateTx(t *testing.T) {
	cState := getState()

	pubkey := createTestCandidate(cState)

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)

	coin := types.GetBaseCoinID()

	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	value := helpers.BipToPip(big.NewInt(100))

	data := DelegateData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeDelegate,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)

	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	targetBalance, _ := big.NewInt(0).SetString("999899800000000000000000", 10)
	balance := cState.Accounts.GetBalance(addr, coin)
	if balance.Cmp(targetBalance) != 0 {
		t.Fatalf("Target %s balance is not correct. Expected %s, got %s", coin, targetBalance, balance)
	}

	cState.Candidates.RecalculateStakes(109000)

	stake := cState.Candidates.GetStakeOfAddress(pubkey, addr, coin)

	if stake == nil {
		t.Fatalf("Stake not found")
	}

	if stake.Value.Cmp(value) != 0 {
		t.Fatalf("Stake value is not corrent. Expected %s, got %s", value, stake.Value)
	}
}

func TestDelegateTxWithWatchlist(t *testing.T) {
	cState := getState()
	pubkey := createTestCandidate(cState)
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	coin := types.GetBaseCoinID()
	value := helpers.BipToPip(big.NewInt(100))
	watchlistAmount := helpers.BipToPip(big.NewInt(1000))

	cState.Watchlist.AddWatchList(addr, pubkey, coin, watchlistAmount)
	cState.Accounts.AddBalance(addr, coin, helpers.BipToPip(big.NewInt(1000000)))

	data := DelegateData{
		PubKey: pubkey,
		Coin:   coin,
		Value:  value,
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
		Type:          TypeDelegate,
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

	response := RunTx(cState, encodedTx, big.NewInt(0), 0, &sync.Map{}, 0)
	if response.Code != 0 {
		t.Fatalf("Response code is not 0. Error %s", response.Log)
	}

	cState.Candidates.RecalculateStakes(109000)
	stake := cState.Candidates.GetStakeOfAddress(pubkey, addr, coin)
	if stake == nil {
		t.Fatalf("Stake not found")
	}

	amount := new(big.Int).Add(value, watchlistAmount)
	if stake.Value.Cmp(amount) != 0 {
		t.Fatalf("Stake value is not corrent. Expected %s, got %s", amount, stake.Value)
	}

	wl := cState.Watchlist.Get(addr, pubkey, coin)
	if wl != nil {
		t.Fatalf("Watchlist is not deleted")
	}
}
