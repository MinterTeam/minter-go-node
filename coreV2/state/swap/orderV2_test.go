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

func TestPair_LoadOrders_bagSkip9295610(t *testing.T) {
	//versionWithoutBug = -1
	//defer func() { versionWithoutBug = math.MaxInt64 }()
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := NewV2(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1,
		helpers.StringToBigInt("31023294253238972794128174"),
		helpers.StringToBigInt("58101053094181360477372"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair := swap.Pair(0, 1)

	pair.AddOrder(
		helpers.StringToBigInt("750000000000000000000"),
		helpers.StringToBigInt("1216201722651929964"), types.Address{1}, 8858209) // 3691

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	remove0 := pair.AddOrder(
		helpers.StringToBigInt("500000000000000000000000"),
		helpers.StringToBigInt("899999999999999999999"), types.Address{1}, 9289553) // 4909

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	immutableTree, err = tree.NewMutableTree(3, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = NewV2(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	pair.AddOrder(
		helpers.StringToBigInt("20600000000000000000000"),
		helpers.StringToBigInt("36989101097630229235"), types.Address{1}, 9294610) // 4914

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	removedOrder := pair.AddOrder(
		helpers.StringToBigInt("3060000000000000000000"),
		helpers.StringToBigInt("5496234812874385476"), types.Address{1}, 9294658) // 4916

	t.Log(pair.sellOrders.ids)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)

	}
	swap.PairRemoveLimitOrder(remove0.ID())
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair.calculateBuyForSellWithOrders(helpers.StringToBigInt("10000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair.SellWithOrders(helpers.StringToBigInt("5000000000000000000000"))

	pair.AddOrder(
		helpers.StringToBigInt("1000000000000000000000000"),
		helpers.StringToBigInt("1500000000000000000000"), types.Address{1}, 9295610)

	pair.SellWithOrders(helpers.StringToBigInt("5786289331250006000000"))

	swap.PairRemoveLimitOrder(removedOrder.ID())

	var v int64
	_, v, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	index0Cache := pair.orderSellByIndex(0)

	immutableTree, err = tree.NewMutableTree(uint64(v), memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = NewV2(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	index0DB := pair.orderSellByIndex(0)
	t.Log(index0DB.id, index0Cache.id)
	if index0DB.id != index0Cache.id {
		t.Error("bug", index0DB.id, index0Cache.id)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
}

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
