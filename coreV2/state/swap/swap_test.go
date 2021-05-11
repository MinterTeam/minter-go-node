package swap

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"testing"
)

func TestPair_load(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
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
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
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

func TestPair_SetOrdersHigher(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	r0 := big.NewInt(1)
	r1 := big.NewInt(9e18)
	_, _, _, _ = swap.PairCreate(0, 1, r0, r1)
	pair := swap.Pair(0, 1)

	pair.SetHigherOrder(big.NewInt(9e18), big.NewInt(1e17+50))
	pair.SetHigherOrder(big.NewInt(7e2), big.NewInt(9e18+50))
	pair.SetHigherOrder(big.NewInt(2e18), big.NewInt(1e17+50))
	pair.SetHigherOrder(big.NewInt(2e18), big.NewInt(1e16+50))
	pair.SetHigherOrder(big.NewInt(1000), big.NewInt(3))
	pair.SetHigherOrder(big.NewInt(100), big.NewInt(3))
	pair.SetHigherOrder(big.NewInt(1), big.NewInt(99))
	pair.SetHigherOrder(big.NewInt(3), big.NewInt(99))
	pair.SetHigherOrder(big.NewInt(5), big.NewInt(99))
	pair.SetHigherOrder(big.NewInt(1e18), big.NewInt(33))
	pair.SetHigherOrder(big.NewInt(100), big.NewInt(33))
	pair.SetHigherOrder(big.NewInt(1e17), big.NewInt(3e17+1))
	pair.SetHigherOrder(big.NewInt(1e17), big.NewInt(3e17))
	pair.SetHigherOrder(big.NewInt(1e17), big.NewInt(3e17+2))
	pair.SetHigherOrder(big.NewInt(1e17), big.NewInt(2e17))
	pair.SetHigherOrder(big.NewInt(1e17), big.NewInt(5e17))
	pair.SetHigherOrder(big.NewInt(1e17), big.NewInt(4e17))
	pair.SetHigherOrder(big.NewInt(2e17), big.NewInt(1e17))

	pair.SetHigherOrder(big.NewInt(1e18), big.NewInt(11))
	pair.SetHigherOrder(big.NewInt(1e18), big.NewInt(10))
	pair.SetHigherOrder(big.NewInt(1e18), big.NewInt(9))

	// r := rand.New(rand.NewSource(5))
	// for i := int64(1); i <= 1000; i++ {
	// 	valueSell := big.NewInt(1e17)
	// 	valueBuy := big.NewInt(0).Add(big.NewInt(1e17), big.NewInt(0).Rand(r, big.NewInt(1e17)))
	// 	pair.SetHigherOrder(valueSell, valueBuy)
	// }
	// for i := int64(1); i <= 1000; i++ {
	// 	valueSell := big.NewInt(0).Add(big.NewInt(1e18), big.NewInt(0).Rand(r, big.NewInt(1e18)))
	// 	valueBuy := big.NewInt(1e17)
	// 	pair.SetHigherOrder(valueSell, valueBuy)
	// }

	for i, limit := range pair.ordersHigher {
		// t.Log(limit.id, limit.Rate().Text('g', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.ordersHigher[i-1]
		if limit.Rate().Cmp(prev.Rate()) != 1 && limit.id < prev.id {
			t.Errorf("not reversed: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}
	count := len(pair.ordersHigher)
	lastID := pair.ordersHigher[(count - 1)].id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	pair = swap.Pair(0, 1)
	last, index := pair.OrderLowerLast()
	if index != -1 {
		t.Error("has orders below course", index)
	}
	if last != nil {
		t.Error("has orders below course", last.id)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	pair = swap.Pair(0, 1)
	last, index = pair.OrderHigherLast()

	if index != (count - 1) {
		t.Fatal(index)
	}
	if last.id != lastID {
		t.Fatal(last.id)
	}

	for i, limit := range pair.ordersHigher {
		// t.Log(limit.id, limit.Rate().Text('f', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.ordersHigher[i-1]
		if limit.Rate().Cmp(prev.Rate()) == -1 || (limit.Rate().Cmp(prev.Rate()) == 0 && limit.id < prev.id) {
			t.Errorf("not reversed: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}

	// reverse x2 pair and sell to buy
	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	last, index = pair.OrderLowerLast()

	if index != (count - 1) {
		t.Fatal("error:", index, count-1)
	}
	if last.id != lastID {
		t.Fatal("error:", last.id, lastID)
	}

	for i, limit := range pair.ordersLower {
		// t.Log(limit.id, limit.Rate().Text('f', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.ordersLower[i-1]
		if limit.Rate().Cmp(prev.Rate()) == 1 || (limit.Rate().Cmp(prev.Rate()) == 0 && limit.id < prev.id) {
			t.Errorf("not reversed: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}

	// reverse pair
	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	last, index = pair.OrderHigherLast()
	if index != -1 {
		t.Error("has orders below course", index)
	}
	if last != nil {
		t.Error("has orders below course", last.id)
	}

}

func TestPair_SetOrderLower(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	r0 := big.NewInt(1)
	r1 := big.NewInt(9e18)
	_, _, _, _ = swap.PairCreate(0, 1, r0, r1)
	pair := swap.Pair(0, 1)

	pair.SetLowerOrder(big.NewInt(9e18), big.NewInt(1e17+50))
	pair.SetLowerOrder(big.NewInt(7e2), big.NewInt(9e18+50))
	pair.SetLowerOrder(big.NewInt(2e18), big.NewInt(1e17+50))
	pair.SetLowerOrder(big.NewInt(2e18), big.NewInt(1e16+50))
	pair.SetLowerOrder(big.NewInt(1000), big.NewInt(3))
	pair.SetLowerOrder(big.NewInt(100), big.NewInt(3))
	pair.SetLowerOrder(big.NewInt(1), big.NewInt(99))
	pair.SetLowerOrder(big.NewInt(3), big.NewInt(99))
	pair.SetLowerOrder(big.NewInt(5), big.NewInt(99))
	pair.SetLowerOrder(big.NewInt(1e18), big.NewInt(33))
	pair.SetLowerOrder(big.NewInt(100), big.NewInt(33))
	pair.SetLowerOrder(big.NewInt(1e17), big.NewInt(3e17+1))
	pair.SetLowerOrder(big.NewInt(1e17), big.NewInt(3e17))
	pair.SetLowerOrder(big.NewInt(1e17), big.NewInt(3e17+2))
	pair.SetLowerOrder(big.NewInt(1e17), big.NewInt(2e17))
	pair.SetLowerOrder(big.NewInt(1e17), big.NewInt(5e17))
	pair.SetLowerOrder(big.NewInt(1e17), big.NewInt(4e17))
	pair.SetLowerOrder(big.NewInt(2e17), big.NewInt(1e17))

	pair.SetLowerOrder(big.NewInt(1e18), big.NewInt(11))
	pair.SetLowerOrder(big.NewInt(1e18), big.NewInt(10))
	pair.SetLowerOrder(big.NewInt(1e18), big.NewInt(9))

	// r := rand.New(rand.NewSource(5))
	// for i := int64(1); i <= 1000; i++ {
	// 	valueSell := big.NewInt(1e17)
	// 	valueBuy := big.NewInt(0).Add(big.NewInt(1e17), big.NewInt(0).Rand(r, big.NewInt(1e17)))
	// 	pair.SetLowerOrder(valueSell, valueBuy)
	// }
	// for i := int64(1); i <= 1000; i++ {
	// 	valueSell := big.NewInt(0).Add(big.NewInt(1e18), big.NewInt(0).Rand(r, big.NewInt(1e18)))
	// 	valueBuy := big.NewInt(1e17)
	// 	pair.SetLowerOrder(valueSell, valueBuy)
	// }

	for i, limit := range pair.ordersLower {
		// t.Log(limit.id, limit.Rate().Text('g', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.ordersLower[i-1]
		if limit.Rate().Cmp(prev.Rate()) != 1 && limit.id < prev.id {
			t.Errorf("not reversed: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}
	count := len(pair.ordersLower)
	lastID := pair.ordersLower[(count - 1)].id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	pair = swap.Pair(0, 1)
	last, index := pair.OrderLowerLast()
	if index != -1 {
		t.Error("has orders higher course", index)
	}
	if last != nil {
		t.Error("has orders higher course", last.id)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	pair = swap.Pair(0, 1)
	last, index = pair.OrderHigherLast()

	if index != (count - 1) {
		t.Fatal(index)
	}
	if last.id != lastID {
		t.Fatal(last.id, lastID)
	}

	for i, limit := range pair.ordersHigher {
		// t.Log(limit.id, limit.Rate().Text('f', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.ordersHigher[i-1]
		if limit.Rate().Cmp(prev.Rate()) == -1 || (limit.Rate().Cmp(prev.Rate()) == 0 && limit.id < prev.id) {
			t.Errorf("not reversed: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}

	// reverse x2 pair and sell to buy
	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	last, index = pair.OrderLowerLast()

	if index != (count - 1) {
		t.Fatal("error:", index, count-1)
	}
	if last.id != lastID {
		t.Fatal("error:", last.id, lastID)
	}

	for i, limit := range pair.ordersLower {
		// t.Log(limit.id, limit.Rate().Text('f', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.ordersLower[i-1]
		if limit.Rate().Cmp(prev.Rate()) == 1 || (limit.Rate().Cmp(prev.Rate()) == 0 && limit.id < prev.id) {
			t.Errorf("not reversed: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}

	// reverse pair
	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	last, index = pair.OrderHigherLast()
	if index != -1 {
		t.Error("has orders below course", index)
	}
	if last != nil {
		t.Error("has orders below course", last.id)
	}

}

func TestPair_SetOrder_sortedPair_lowerAndHigher(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())

	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(9e18), big.NewInt(9e18))
	pair := swap.Pair(0, 1)

	idLower := pair.SetOrder(big.NewInt(1e17), big.NewInt(9e17))
	if idLower != pair.ordersLower[0].id {
		t.Error("error set lower order price")
	}

	lastLower, indexLower := pair.OrderLowerLast()
	if len(pair.ordersLower)-1 != indexLower {
		t.Error("count lower orders price is not 1")
	}
	if idLower != lastLower.id {
		t.Error("error load lower order price")
	}
	if idLower != pair.ordersLower[0].id {
		t.Error("error load lower order price")
	}

	idHigher := pair.SetOrder(big.NewInt(9e17), big.NewInt(1e17))
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error set higher order price")
	}

	lastHigher, indexHigher := pair.OrderHigherLast()
	if len(pair.ordersHigher)-1 != indexHigher {
		t.Error("count higher orders price is not 1")
	}
	if idHigher != lastHigher.id {
		t.Error("error load higher order price")
	}
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error load higher order price")
	}

	if lastLower.Rate().Cmp(pair.Rate()) != -1 {
		t.Error("not ordered lower and current")
	}
	if lastHigher.Rate().Cmp(pair.Rate()) != 1 {
		t.Error("not ordered higher and current")
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)
	lastHigher, indexHigher = pair.OrderHigherLast()
	if indexHigher != 0 {
		t.Fatal("count higher orders price is not 1", indexHigher)
	}
	if len(pair.ordersHigher)-1 != indexHigher {
		t.Error("count higher orders price is not 1")
	}
	if idHigher != lastHigher.id {
		t.Error("error load higher order price")
	}
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error load higher order price")
	}

	lastLower, indexLower = pair.OrderLowerLast()
	if indexLower != 0 {
		t.Fatal("count lower orders price is not 1", indexLower)
	}
	if len(pair.ordersLower)-1 != indexLower {
		t.Error("count lower orders price is not 1")
	}
	if idLower != lastLower.id {
		t.Error("error load lower order price")
	}
	if idLower != pair.ordersLower[0].id {
		t.Error("error load lower order price")
	}
}

func TestPair_SetOrder_cmpUnsortedCommit(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())

	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(9e18), big.NewInt(9e18))
	pair := swap.Pair(0, 1)
	_ = pair.SetOrder(big.NewInt(9e17), big.NewInt(1e17))
	_ = pair.SetOrder(big.NewInt(9e17), big.NewInt(1e17))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	memDB1 := db.NewMemDB()
	immutableTree1, err := tree.NewMutableTree(0, memDB1, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus1 := bus.NewBus()
	checker.NewChecker(newBus1)

	swap1 := New(newBus, immutableTree1.GetLastImmutable())
	_, _, _, _ = swap1.PairCreate(0, 1, big.NewInt(9e18), big.NewInt(9e18))
	pair1 := swap1.Pair(0, 1)
	_ = pair1.SetOrder(big.NewInt(9e17), big.NewInt(1e17))
	_ = pair1.SetOrder(big.NewInt(9e17), big.NewInt(1e17))

	_, _, err = immutableTree1.Commit(swap1)
	if err != nil {
		t.Fatal(err)
	}

	if memDB.Stats()["database.size"] != memDB1.Stats()["database.size"] {
		t.Log("diff size")
	}
}

func TestPair_SetOrder_unsortedPair(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())

	_, _, _, _ = swap.PairCreate(1, 0, big.NewInt(9e18), big.NewInt(9e18))
	pair := swap.Pair(1, 0)

	idLower := pair.SetOrder(big.NewInt(1e17), big.NewInt(9e17))
	if idLower != pair.ordersLower[0].id {
		t.Error("error set lower order price")
	}

	lastLower, indexLower := pair.OrderLowerLast()
	if len(pair.ordersLower)-1 != indexLower {
		t.Error("count lower orders price is not 1")
	}
	if idLower != lastLower.id {
		t.Error("error load lower order price")
	}
	if idLower != pair.ordersLower[0].id {
		t.Error("error load lower order price")
	}

	idHigher := pair.SetOrder(big.NewInt(9e17), big.NewInt(1e17))
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error set higher order price")
	}

	lastHigher, indexHigher := pair.OrderHigherLast()
	if len(pair.ordersHigher)-1 != indexHigher {
		t.Error("count higher orders price is not 1")
	}
	if idHigher != lastHigher.id {
		t.Error("error load higher order price")
	}
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error load higher order price")
	}

	if lastLower.Rate().Cmp(pair.Rate()) != -1 {
		t.Error("not ordered lower and current")
	}
	if lastHigher.Rate().Cmp(pair.Rate()) != 1 {
		t.Error("not ordered higher and current")
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	lastHigher, indexHigher = pair.OrderHigherLast()
	if indexHigher != 0 {
		t.Fatal("count higher orders price is not 1", indexHigher)
	}
	if len(pair.ordersHigher)-1 != indexHigher {
		t.Error("count higher orders price is not 1")
	}
	if idHigher != lastHigher.id {
		t.Error("error load higher order price")
	}
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error load higher order price")
	}

	lastLower, indexLower = pair.OrderLowerLast()
	if indexLower != 0 {
		t.Fatal("count lower orders price is not 1", indexLower)
	}
	if len(pair.ordersLower)-1 != indexLower {
		t.Error("count lower orders price is not 1")
	}
	if idLower != lastLower.id {
		t.Error("error load lower order price")
	}
	if idLower != pair.ordersLower[0].id {
		t.Error("error load lower order price")
	}
}

func TestPair_SetOrder_reversePairLoad(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())

	_, _, _, _ = swap.PairCreate(1, 0, big.NewInt(9e18), big.NewInt(9e18))
	pair := swap.Pair(1, 0)

	idHigher := pair.SetOrder(big.NewInt(1e17), big.NewInt(9e17))
	idLower := pair.SetOrder(big.NewInt(9e17), big.NewInt(1e17))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	idHigher, idLower = idLower, idHigher

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	lastHigher, indexHigher := pair.OrderHigherLast()
	if indexHigher != 0 {
		t.Fatal("count higher orders price is not 1", indexHigher)
	}
	if len(pair.ordersHigher)-1 != indexHigher {
		t.Error("count higher orders price is not 1")
	}
	if idHigher != lastHigher.id {
		t.Error("error load higher order price")
	}
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error load higher order price")
	}

	lastLower, indexLower := pair.OrderLowerLast()
	if indexLower != 0 {
		t.Fatal("count lower orders price is not 1", indexLower)
	}
	if len(pair.ordersLower)-1 != indexLower {
		t.Error("count lower orders price is not 1")
	}
	if idLower != lastLower.id {
		t.Error("error load lower order price")
	}
	if idLower != pair.ordersLower[0].id {
		t.Error("error load lower order price")
	}
}

func TestPair_SetOrder_reversePairSet(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())

	_, _, _, _ = swap.PairCreate(1, 0, big.NewInt(9e18), big.NewInt(9e18))
	pair := swap.Pair(0, 1)

	idLower := pair.SetOrder(big.NewInt(1e17), big.NewInt(9e17))
	idHigher := pair.SetOrder(big.NewInt(9e17), big.NewInt(1e17))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	idHigher, idLower = idLower, idHigher

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(1, 0)
	lastHigher, indexHigher := pair.OrderHigherLast()
	if indexHigher != 0 {
		t.Fatal("count higher orders price is not 1", indexHigher)
	}
	if len(pair.ordersHigher)-1 != indexHigher {
		t.Error("count higher orders price is not 1")
	}
	if idHigher != lastHigher.id {
		t.Error("error load higher order price")
	}
	if idHigher != pair.ordersHigher[0].id {
		t.Error("error load higher order price")
	}

	lastLower, indexLower := pair.OrderLowerLast()
	if indexLower != 0 {
		t.Fatal("count lower orders price is not 1", indexLower)
	}
	if len(pair.ordersLower)-1 != indexLower {
		t.Error("count lower orders price is not 1")
	}
	if idLower != lastLower.id {
		t.Error("error load lower order price")
	}
	if idLower != pair.ordersLower[0].id {
		t.Error("error load lower order price")
	}
}
