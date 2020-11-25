package halts

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"testing"
)

func TestHaltsToDeleteModel(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	h := NewHalts(bus.NewBus(), mutableTree.GetLastImmutable())

	pubkey, height := types.Pubkey{0}, uint64(10)

	h.AddHaltBlock(height, pubkey)
	if err := h.Commit(mutableTree.MutableTree()); err != nil {
		t.Fatal(err)
	}

	_, _, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}
	h.SetImmutableTree(mutableTree.GetLastImmutable())

	if h.GetHaltBlocks(height) == nil {
		t.Fatal("Halts not found")
	}

	h.Delete(height)
	if err := h.Commit(mutableTree.MutableTree()); err != nil {
		t.Fatal(err)
	}

	_, _, err = mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}
	h.SetImmutableTree(mutableTree.GetLastImmutable())

	if h.GetHaltBlocks(height) != nil {
		t.Fatal("Halts not deleted")
	}
}

func TestBusToAddHaltBlock(t *testing.T) {
	mutableTree, _ := tree.NewMutableTree(0, db.NewMemDB(), 1024)
	h := NewHalts(bus.NewBus(), mutableTree.GetLastImmutable())

	pubkey, height := types.Pubkey{0}, uint64(10)

	hbBus := Bus{halts: h}
	hbBus.AddHaltBlock(height, pubkey)

	if err := h.Commit(mutableTree.MutableTree()); err != nil {
		t.Fatal(err)
	}

	_, _, err := mutableTree.SaveVersion()
	if err != nil {
		t.Fatal(err)
	}

	h.SetImmutableTree(mutableTree.GetLastImmutable())

	halt := h.GetHaltBlocks(height)
	if halt == nil {
		t.Fatal("Halts not found")
	}

	if len(halt.List) != 1 {
		t.Fatalf("Incorrect amount of halts: %d. Expected: 1", len(halt.List))
	}

	if halt.Height() != height {
		t.Fatalf("Invalid height %d. Expected %d", halt.Height(), height)
	}

	hbPubKey := halt.List[0].Pubkey
	if !hbPubKey.Equals(pubkey) {
		t.Fatalf("Invalid public key %s. Expected %s", hbPubKey.String(), pubkey.String())
	}
}
