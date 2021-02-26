package frozenfunds

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/treeV2"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestFrozenFundsToAddModel(t *testing.T) {
	t.Parallel()
	b := bus.NewBus()
	mutableTree, _ := treeV2.NewMutableTree(0, db.NewMemDB(), 1024, 0)

	ff := NewFrozenFunds(b, mutableTree.GetLastImmutable())

	b.SetChecker(checker.NewChecker(b))
	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	b.SetCoins(coins.NewBus(coinsState))

	height, addr, pubkey, coin, val := uint64(1), types.Address{0}, types.Pubkey{0}, types.GetBaseCoinID(), big.NewInt(1e18)

	ff.AddFund(height, addr, pubkey, 1, coin, val, nil)

	_, _, err := mutableTree.Commit(ff)
	if err != nil {
		t.Fatal(err)
	}

	funds := ff.GetFrozenFunds(height)
	if funds == nil {
		t.Fatal("Funds not found")
	}

	if len(funds.List) != 1 {
		t.Fatal("Incorrect amount of funds")
	}

	if funds.Height() != height {
		t.Fatal("Invalid funds data")
	}

	f := funds.List[0]
	if !pubkey.Equals(*f.CandidateKey) || f.Value.Cmp(val) != 0 || f.Address.Compare(addr) != 0 || f.Coin != coin {
		t.Fatal("Invalid funds data")
	}
}

func TestFrozenFundsToDeleteModel(t *testing.T) {
	t.Parallel()
	b := bus.NewBus()
	mutableTree, _ := treeV2.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	ff := NewFrozenFunds(b, mutableTree.GetLastImmutable())

	b.SetChecker(checker.NewChecker(b))
	coinsState := coins.NewCoins(b, mutableTree.GetLastImmutable())

	b.SetCoins(coins.NewBus(coinsState))

	height, addr, pubkey, coin, val := uint64(1), types.Address{0}, types.Pubkey{0}, types.GetBaseCoinID(), big.NewInt(1e18)

	ff.AddFund(height, addr, pubkey, 1, coin, val, nil)

	_, _, err := mutableTree.Commit(ff)
	if err != nil {
		t.Fatal(err)
	}

	if funds := ff.GetFrozenFunds(height); funds == nil {
		t.Fatal("Funds not found")
	}

	ff.Delete(height)

	_, _, err = mutableTree.Commit(ff)
	if err != nil {
		t.Fatal(err)
	}

	if funds := ff.GetFrozenFunds(height); funds != nil {
		t.Fatal("Funds not deleted")
	}
}

func TestFrozenFundsToDeleteNotExistingFund(t *testing.T) {
	t.Parallel()
	b := bus.NewBus()
	mutableTree, _ := treeV2.NewMutableTree(0, db.NewMemDB(), 1024, 0)
	ff := NewFrozenFunds(b, mutableTree.GetLastImmutable())

	ff.Delete(0)
}
