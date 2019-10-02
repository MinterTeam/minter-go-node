package old

import (
	"bytes"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func getState() *StateDB {
	s, err := New(0, db.NewMemDB(), false)
	if err != nil {
		panic(err)
	}

	return s
}

func TestStateDB_AddBalance(t *testing.T) {
	state := getState()

	address := types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb")

	balance := state.GetBalance(address, types.GetBaseCoin())
	if balance.Cmp(types.Big0) != 0 {
		t.Errorf("Balance of %s should be 0, got %s", address.String(), balance)
	}

	newBalance := helpers.BipToPip(big.NewInt(10))
	state.AddBalance(address, types.GetBaseCoin(), newBalance)

	balance = state.GetBalance(address, types.GetBaseCoin())
	if balance.Cmp(newBalance) != 0 {
		t.Errorf("Balance of %s should be %s, got %s", address.String(), newBalance, balance)
	}
}

func TestStateDB_SubBalance(t *testing.T) {
	state := getState()

	address := types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb")

	initialBalance := helpers.BipToPip(big.NewInt(10))
	state.SetBalance(address, types.GetBaseCoin(), initialBalance)

	balance := state.GetBalance(address, types.GetBaseCoin())
	if balance.Cmp(initialBalance) != 0 {
		t.Errorf("Balance of %s should be %s, got %s", address.String(), initialBalance, balance)
	}

	amount := helpers.BipToPip(big.NewInt(10))
	state.SubBalance(address, types.GetBaseCoin(), amount)

	balance = state.GetBalance(address, types.GetBaseCoin())
	target := big.NewInt(0).Sub(initialBalance, amount)
	if balance.Cmp(target) != 0 {
		t.Errorf("Balance of %s should be %s, got %s", address.String(), target, balance)
	}
}

func TestStateDB_SetNonce(t *testing.T) {
	state := getState()

	address := types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb")

	nonce := state.GetNonce(address)
	if nonce != 0 {
		t.Errorf("Nonce of %s should be 0, got %d", address.String(), nonce)
	}

	newNonce := uint64(1)
	state.SetNonce(address, newNonce)

	nonce = state.GetNonce(address)
	if nonce != newNonce {
		t.Errorf("Nonce of %s should be %d, got %d", address.String(), newNonce, nonce)
	}
}

func TestStateDB_Commit(t *testing.T) {
	state := getState()
	state.AddBalance(types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb"), types.GetBaseCoin(),
		big.NewInt(1))

	symbol := types.CoinSymbol{}
	copy(symbol[:], []byte("TEST"))
	state.CreateCoin(symbol, "TEST NAME", big.NewInt(10), 10, big.NewInt(10))

	ff := state.GetOrNewStateFrozenFunds(2)
	ff.AddFund(types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb"), []byte{}, types.GetBaseCoin(),
		big.NewInt(2))

	hash, version, err := state.Commit()
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}

	if version != 1 {
		t.Errorf("Version should be 1, got %d", version)
	}

	targetHash, _ := hex.DecodeString("466e1c2aca40c0db51e54f6d87224c2eede66b26b96fa1f4c762d66a3d93e637,")
	if !bytes.Equal(hash, targetHash) {
		t.Errorf("Hash should be %x, got %x", targetHash, hash)
	}
}

func TestStateDB_GetBalances(t *testing.T) {
	state := getState()

	address := types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb")
	newBalance := helpers.BipToPip(big.NewInt(10))
	state.AddBalance(address, types.GetBaseCoin(), newBalance)

	expect := Balances{
		Data: map[types.CoinSymbol]*big.Int{types.GetBaseCoin(): newBalance},
	}

	balances := state.GetBalances(address)
	if len(balances.Data) != len(expect.Data) ||
		balances.Data[types.GetBaseCoin()].Cmp(expect.Data[types.GetBaseCoin()]) != 0 {
		t.Errorf("Balances of %s are not like expected", address.String())
	}
}

func TestStateDB_GetEmptyBalances(t *testing.T) {
	state := getState()

	address := types.HexToAddress("Mx02003587993aba5276925c058ba082d209e61cbb")
	balances := state.GetBalances(address)
	if len(balances.Data) != 0 {
		t.Errorf("Balances of %s are not like expected", address.String())
	}
}
