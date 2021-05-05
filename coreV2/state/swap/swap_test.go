package swap

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"math/rand"
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

func TestPair_SetOrderSell(t *testing.T) {
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
	pair := swap.Pair(0, 1)

	pair.SetOrderSell(big.NewInt(9e18), big.NewInt(1e17+50))
	pair.SetOrderSell(big.NewInt(7e2), big.NewInt(9e18+50))
	pair.SetOrderSell(big.NewInt(2e18), big.NewInt(1e17+50))
	pair.SetOrderSell(big.NewInt(2e18), big.NewInt(1e16+50))
	pair.SetOrderSell(big.NewInt(1000), big.NewInt(3))
	pair.SetOrderSell(big.NewInt(100), big.NewInt(3))
	pair.SetOrderSell(big.NewInt(1), big.NewInt(99))
	pair.SetOrderSell(big.NewInt(3), big.NewInt(99))
	pair.SetOrderSell(big.NewInt(5), big.NewInt(99))
	pair.SetOrderSell(big.NewInt(1e18), big.NewInt(33))
	pair.SetOrderSell(big.NewInt(100), big.NewInt(33))
	pair.SetOrderSell(big.NewInt(1e17), big.NewInt(3e17+1))
	pair.SetOrderSell(big.NewInt(1e17), big.NewInt(3e17))
	pair.SetOrderSell(big.NewInt(1e17), big.NewInt(3e17+2))
	pair.SetOrderSell(big.NewInt(1e17), big.NewInt(2e17))
	pair.SetOrderSell(big.NewInt(1e17), big.NewInt(5e17))
	pair.SetOrderSell(big.NewInt(1e17), big.NewInt(4e17))
	pair.SetOrderSell(big.NewInt(2e17), big.NewInt(1e17))

	pair.SetOrderSell(big.NewInt(1e18), big.NewInt(11))
	pair.SetOrderSell(big.NewInt(1e18), big.NewInt(10))
	pair.SetOrderSell(big.NewInt(1e18), big.NewInt(9))

	r := rand.New(rand.NewSource(5))
	for i := int64(1); i <= 1000; i++ {
		valueSell := big.NewInt(1e17)
		valueBuy := big.NewInt(0).Add(big.NewInt(1e17), big.NewInt(0).Rand(r, big.NewInt(1e17)))
		pair.SetOrderSell(valueSell, valueBuy)
	}
	for i := int64(1); i <= 1000; i++ {
		valueSell := big.NewInt(0).Add(big.NewInt(1e18), big.NewInt(0).Rand(r, big.NewInt(1e18)))
		valueBuy := big.NewInt(1e17)
		pair.SetOrderSell(valueSell, valueBuy)
	}

	for i, limit := range pair.limitsSell {
		// t.Log(limit.id, limit.Rate().Text('g', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.limitsSell[i-1]
		if limit.Rate().Cmp(prev.Rate()) != 1 && limit.id < prev.id {
			t.Errorf("not sorted: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}
	count := len(pair.limitsSell)
	lastID := pair.limitsSell[(count - 1)].id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	pair = swap.Pair(0, 1)
	last, index := pair.OrderSellLast()

	if index != (count - 1) {
		t.Error(index)
	}
	if last.id != lastID {
		t.Error(last.id)
	}

	for i, limit := range pair.limitsSell {
		// t.Log(limit.id, limit.Rate().Text('g', 18), limit.Coin0, limit.Coin1)
		if i == 0 {
			continue
		}
		prev := pair.limitsSell[i-1]
		if limit.Rate().Cmp(prev.Rate()) == -1 || (limit.Rate().Cmp(prev.Rate()) == 0 && limit.id < prev.id) {
			t.Errorf("not sorted: [%d]%v < [%d]%v", i, limit.Rate(), i-1, prev.Rate())
		}
	}
}
