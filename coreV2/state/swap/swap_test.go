package swap

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/treeV2"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestPair_load(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := treeV2.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	r0 := big.NewInt(1e18)
	r2 := big.NewInt(2e18)
	swap.PairCreate(0, 2, r0, r2)
	pair := swap.Pair(0, 2)
	if pair == nil {
		t.Fatal("not found")
	}
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	pair = swap.Pair(2, 0)
	if pair == nil {
		t.Fatal("not found")
	}
	if pair.Reserve0.Cmp(r2) != 0 {
		t.Fatal("r2")
	}
	if pair.Reserve1.Cmp(r0) != 0 {
		t.Fatal("r0")
	}
	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 2)
	if pair == nil {
		t.Fatal("not found")
	}

	if pair.Reserve0.Cmp(r0) != 0 {
		t.Fatal("r0")
	}
	if pair.Reserve1.Cmp(r2) != 0 {
		t.Fatal("r2")
	}
	pair = swap.Pair(2, 0)
	if pair == nil {
		t.Fatal("not found")
	}
	if pair.Reserve0.Cmp(r2) != 0 {
		t.Fatal("r2")
	}
	if pair.Reserve1.Cmp(r0) != 0 {
		t.Fatal("r0")
	}
}

func TestPair_commission(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := treeV2.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	r0 := big.NewInt(1e18)
	r1 := big.NewInt(1e18)
	_, _, _, _ = swap.PairCreate(0, 1, r0, r1)

	valueSwap := big.NewInt(1e17)
	{
		swap.PairBuy(0, 1, big.NewInt(1e18), valueSwap)
		// _ = swap.Pair(0, 1)
		// t.Log(pair.Reserves())
	}

	{
		swap.PairSell(1, 0, valueSwap, big.NewInt(0))
		// _ = swap.Pair(0, 1)
		// t.Log(pair.Reserves())
	}

}
