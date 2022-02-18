package swap

import (
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"math/big"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
)

func TestSwap_PairSellSimple(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	accounts.NewBus(accounts.NewAccounts(newBus, immutableTree.GetLastImmutable()))
	events := &eventsdb.MockEvents{}
	newBus.SetEvents(events)

	swap := NewV2(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.BipToPip(big.NewInt(1)), helpers.BipToPip(big.NewInt(1)))
	pair := swap.Pair(0, 1)
	pair.AddOrder(helpers.BipToPip(big.NewInt(1001)), helpers.BipToPip(big.NewInt(999)), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	swap = NewV2(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)
	t.Log(pair.BuyWithOrders(big.NewInt(0).Add(helpers.BipToPip(big.NewInt(998)), big.NewInt(2e15))))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", events.LoadEvents(0))
}
