package accounts

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestAccounts_CreateMultisig(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())

	multisigAddr := accounts.CreateMultisig([]uint32{1, 1, 2}, []types.Address{[20]byte{1}, [20]byte{2}, [20]byte{3}}, 2, [20]byte{4})

	account := accounts.GetAccount(multisigAddr)
	if account == nil {
		t.Fatal("account is nil")
	}

	if !account.IsMultisig() {
		t.Fatal("account is not multisig")
	}

	multisig := account.Multisig()
	if multisig.GetWeight([20]byte{1, 1, 2, 3, 4, 5}) != 0 {
		t.Fatal("address weight not equal 0")
	}
	if multisig.GetWeight([20]byte{1}) != 1 {
		t.Fatal("address weight not equal 1")
	}
	if multisig.GetWeight([20]byte{2}) != 1 {
		t.Fatal("address weight not equal 1")
	}
	if multisig.GetWeight([20]byte{3}) != 2 {
		t.Fatal("address weight not equal 2")
	}
	if multisig.Threshold != 2 {
		t.Fatal("threshold not equal 2")
	}
}

func TestAccounts_SetNonce(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetNonce([20]byte{4}, 5)
	if accounts.GetNonce([20]byte{4}) != 5 {
		t.Fatal("nonce not equal 5")
	}
}

func TestAccounts_SetBalance(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))
	account := accounts.GetAccount([20]byte{4})
	if account == nil {
		t.Fatal("account is nil")
	}
	if account.getBalance(0).String() != "1000" {
		t.Fatal("balance of coin ID '0' not equal 1000")
	}
}

func TestAccounts_SetBalance_fromDB(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))

	_, _, err := mutableTree.Commit(accounts)
	if err != nil {
		t.Fatal(err)
	}
	if accounts.GetBalance([20]byte{4}, 0).String() != "1000" {
		t.Fatal("balance of coin ID '0' not equal 1000")
	}

	accounts = NewAccounts(b, mutableTree.GetLastImmutable())

	if accounts.GetBalance([20]byte{4}, 0).String() != "1000" {
		t.Fatal("balance of coin ID '0' not equal 1000")
	}
}

func TestAccounts_SetBalance_0(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(100))
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(0))
	accounts.SetBalance([20]byte{4}, 1, big.NewInt(0))
	account := accounts.GetAccount([20]byte{4})
	if account == nil {
		t.Fatal("account is nil")
	}
	if accounts.GetBalance([20]byte{4}, 0).String() != "0" {
		t.Fatal("balance of coin ID '0' is not 0")
	}
	if accounts.GetBalance([20]byte{4}, 1).String() != "0" {
		t.Fatal("balance of coin ID '1' is not 0")
	}
}

func TestAccounts_GetBalances(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	busCoins := coins.NewCoins(b, mutableTree.GetLastImmutable())

	b.SetCoins(coins.NewBus(busCoins))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))

	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	coinsState.Create(1,
		types.StrToCoinSymbol("AAA"),
		"AAACOIN",
		helpers.BipToPip(big.NewInt(10)),
		10,
		helpers.BipToPip(big.NewInt(10000)),
		big.NewInt(0).Exp(big.NewInt(10), big.NewInt(10+18), nil),
		nil)

	_, _, err := mutableTree.Commit(accounts)
	if err != nil {
		t.Fatal(err)
	}

	symbol := coinsState.GetCoinBySymbol(types.StrToCoinSymbol("AAA"), 0)
	if symbol == nil {
		t.Fatal("coin not found")
	}

	accounts.SetBalance([20]byte{4}, symbol.ID(), big.NewInt(1001))

	balances := accounts.GetBalances([20]byte{4})
	if len(balances) != 2 {
		t.Fatal("count of coin on balance not equal 2")
	}
	if balances[0].Value.String() != "1000" {
		t.Fatal("balance of coin ID '0' not equal 1000")
	}
	if balances[1].Value.String() != "1001" {
		t.Log(balances[1].Value.String())
		t.Fatal("balance of coin 'AAA' not equal 1001")
	}
}

func TestAccounts_ExistsMultisig(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())

	msigAddress := CreateMultisigAddress([20]byte{4}, 12)
	if accounts.ExistsMultisig(msigAddress) {
		t.Fatal("multisig address is busy")
	}

	accounts.SetBalance(msigAddress, 0, big.NewInt(1))
	if accounts.ExistsMultisig(msigAddress) {
		t.Fatal("multisig address is busy")
	}

	accounts.SetNonce(msigAddress, 1)
	if !accounts.ExistsMultisig(msigAddress) {
		t.Fatal("multisig address is not busy")
	}

	accounts.SetNonce(msigAddress, 0)

	_ = accounts.CreateMultisig([]uint32{1, 1, 2}, []types.Address{[20]byte{1}, [20]byte{2}, [20]byte{3}}, 2, msigAddress)

	if !accounts.ExistsMultisig(msigAddress) {
		t.Fatal("multisig address is free")
	}
}

func TestAccounts_AddBalance_bus(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))

	accounts.bus.Accounts().AddBalance([20]byte{4}, 0, big.NewInt(1000))

	if accounts.GetBalance([20]byte{4}, 0).String() != "2000" {
		t.Fatal("balance of coin ID '0' not equal 2000")
	}
}

func TestAccounts_SubBalance(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))

	accounts.SubBalance([20]byte{4}, 0, big.NewInt(500))

	account := accounts.GetAccount([20]byte{4})
	if account == nil {
		t.Fatal("account is nil")
	}
	if account.getBalance(0).String() != "500" {
		t.Fatal("balance of coin ID '0' not equal 500")
	}
}

func TestAccounts_EditMultisig(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())

	msigAddress := CreateMultisigAddress([20]byte{4}, 12)

	_ = accounts.CreateMultisig([]uint32{3, 3, 6}, []types.Address{[20]byte{1, 1}, [20]byte{2, 3}, [20]byte{3, 3}}, 6, msigAddress)
	_ = accounts.EditMultisig(2, []uint32{1, 1, 2}, []types.Address{[20]byte{1}, [20]byte{2}, [20]byte{3}}, msigAddress)

	account := accounts.GetAccount(msigAddress)
	if account == nil {
		t.Fatal("account is nil")
	}

	if !account.IsMultisig() {
		t.Fatal("account is not multisig")
	}

	multisig := account.Multisig()
	if multisig.GetWeight([20]byte{1}) != 1 {
		t.Fatal("address weight not equal 1")
	}
	if multisig.GetWeight([20]byte{2}) != 1 {
		t.Fatal("address weight not equal 1")
	}
	if multisig.GetWeight([20]byte{3}) != 2 {
		t.Fatal("address weight not equal 2")
	}
	if multisig.Threshold != 2 {
		t.Fatal("threshold not equal 2")
	}
}

func TestAccounts_Commit(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))

	hash, version, err := mutableTree.Commit(accounts)
	if err != nil {
		t.Fatal(err)
	}

	if version != 1 {
		t.Fatalf("version %d", version)
	}

	if fmt.Sprintf("%X", hash) != "FB6DA65ECF998BC4050192B95E33B3A1B7319E88BD83D434D18F2A28EADC3217" {
		t.Fatalf("hash %X", hash)
	}
}

func TestAccounts_Export(t *testing.T) {
	t.Parallel()
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	b := bus.NewBus()
	busCoins := coins.NewCoins(b, mutableTree.GetLastImmutable())
	b.SetCoins(coins.NewBus(busCoins))
	b.SetChecker(checker.NewChecker(b))
	accounts := NewAccounts(b, mutableTree.GetLastImmutable())
	accounts.SetBalance([20]byte{4}, 0, big.NewInt(1000))

	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	coinsState.Create(1,
		types.StrToCoinSymbol("AAA"),
		"AAACOIN",
		helpers.BipToPip(big.NewInt(10)),
		10,
		helpers.BipToPip(big.NewInt(10000)),
		big.NewInt(0).Exp(big.NewInt(10), big.NewInt(10+18), nil),
		nil)

	_, _, err := mutableTree.Commit(accounts)
	if err != nil {
		t.Fatal(err)
	}
	symbol := coinsState.GetCoinBySymbol(types.StrToCoinSymbol("AAA"), 0)
	if symbol == nil {
		t.Fatal("coin not found")
	}

	accounts.SetBalance([20]byte{4}, symbol.ID(), big.NewInt(1001))
	_ = accounts.CreateMultisig([]uint32{1, 1, 2}, []types.Address{[20]byte{1}, [20]byte{2}, [20]byte{3}}, 2, [20]byte{4})

	_, _, err = mutableTree.Commit(accounts)
	if err != nil {
		t.Fatal(err)
	}
	state := new(types.AppState)
	accounts.Export(state)

	bytes, err := json.Marshal(state.Accounts)
	if err != nil {
		t.Fatal(err)
	}

	valid := "[{\"address\":\"Mx0400000000000000000000000000000000000000\",\"balance\":[{\"value\":\"1000\"},{\"coin\":1,\"value\":\"1001\"}],\"nonce\":0,\"multisig_data\":{\"weights\":[1,1,2],\"threshold\":2,\"addresses\":[\"Mx0100000000000000000000000000000000000000\",\"Mx0200000000000000000000000000000000000000\",\"Mx0300000000000000000000000000000000000000\"]}}]"
	if string(bytes) != valid {
		t.Log(string(bytes))
		t.Log(valid)
		t.Fatal("not equal JSON")
	}
}
