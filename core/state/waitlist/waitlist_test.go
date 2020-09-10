package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestWaitListToGetByAddressAndPubKey(t *testing.T) {
	b := bus.NewBus()
	mutableTree := tree.NewMutableTree(0, db.NewMemDB(), 1024)

	wl, err := NewWaitList(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	candidatesState, err := candidates.NewCandidates(b, mutableTree)
	if err != nil {
		t.Fatal(err)
	}

	addr, pubkey, coin, val := types.Address{0}, types.Pubkey{0}, types.GetBaseCoinID(), big.NewInt(1e18)

	candidatesState.Create(addr, addr, addr, pubkey, 10)

	wl.AddWaitList(addr, pubkey, coin, val)
	if err := wl.Commit(); err != nil {
		t.Fatal(err)
	}

	_, _, err = mutableTree.SaveVersion()
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
