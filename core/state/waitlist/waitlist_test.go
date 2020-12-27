package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/state/checker"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestWaitListToGetByAddressAndPubKey(t *testing.T) {
	t.Parallel()
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)

	wl := NewWaitList(b, mutableTree.GetLastImmutable())

	candidatesState := candidates.NewCandidates(b, mutableTree.GetLastImmutable())

	addr, pubkey, coin, val := types.Address{0}, types.Pubkey{0}, types.GetBaseCoinID(), big.NewInt(1e18)

	candidatesState.Create(addr, addr, addr, pubkey, 10, 0)

	wl.AddWaitList(addr, pubkey, coin, val)
	_, _, err := mutableTree.Commit(wl)
	if err != nil {
		t.Fatal(err)
	}
	items := wl.GetByAddressAndPubKey(addr, pubkey)
	if len(items) != 1 {
		t.Fatal("Incorrect amount of items in waitlist")
	}

	if items[0].Value.Cmp(val) != 0 || items[0].Coin != coin {
		t.Fatal("Invalid waitlist data")
	}

	model := wl.GetByAddress(addr)
	if len(model.List) != 1 {
		t.Fatal("Incorrect amount of items in waitlist")
	}
}

func TestWaitListToPartialDelete(t *testing.T) {
	t.Parallel()
	b := bus.NewBus()
	b.SetChecker(checker.NewChecker(b))
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024, 0)

	wl := NewWaitList(b, mutableTree.GetLastImmutable())

	candidatesState := candidates.NewCandidates(b, mutableTree.GetLastImmutable())

	addr, pubkey, coin, val := types.Address{0}, types.Pubkey{0}, types.GetBaseCoinID(), big.NewInt(1e18)
	candidatesState.Create(addr, addr, addr, pubkey, 10, 0)

	wl.AddWaitList(addr, pubkey, coin, val)
	wl.AddWaitList(addr, pubkey, 1, val)
	wl.AddWaitList(addr, pubkey, 2, val)
	_, _, err := mutableTree.Commit(wl)
	if err != nil {
		t.Fatal(err)
	}
	wl.Delete(addr, pubkey, 0)
	wl.Delete(addr, pubkey, 1)
	wl.AddWaitList(addr, pubkey, 1, big.NewInt(1e17))
	_, _, err = mutableTree.Commit(wl)
	if err != nil {
		t.Fatal(err)
	}

	items := wl.GetByAddressAndPubKey(addr, pubkey)
	if len(items) != 2 {
		t.Fatal("Incorrect amount of items in waitlist")
	}

	if items[1].Value.Cmp(big.NewInt(1e17)) != 0 || items[1].Coin != 1 {
		t.Fatal("Invalid waitlist data")
	}

	if items[0].Value.Cmp(val) != 0 || items[0].Coin != 2 {
		t.Fatal("Invalid waitlist data")
	}
}
