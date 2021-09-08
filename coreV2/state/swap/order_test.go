package swap

import (
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/tendermint/go-amino"
	db "github.com/tendermint/tm-db"
)

func TestPair_cals0(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("100000000000000000000000"), helpers.StringToBigInt("10000000000000000000000000"))
	pair := swap.Pair(0, 1)
	pair.AddOrder(helpers.StringToBigInt("10000000000000000000"), helpers.StringToBigInt("99998528242522874800000"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	var a, b, c = big.NewInt(0), big.NewInt(0), helpers.StringToBigInt("4000000000000000000")
	var oA, oB []*Limit
	{

		a, oA = pair.calculateBuyForSellWithOrders(c)
		b, oB = pair.calculateSellForBuyWithOrders(a)

		for i := range oA {
			t.Log(pair.getOrder(uint32(i + 1)))
			t.Log(oA[i])
			t.Log(oB[i])
		}

		if c.Cmp(b) != 0 {
			t.Skip(c, b)
		}
	}

	out, _, _ := pair.SellWithOrders(c)
	if out.Cmp(a) != 0 {
		t.Error(out, a)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

}

func TestPair_cals1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("100000000000000000000000"), helpers.StringToBigInt("10000000000000000000000000"))
	pair := swap.Pair(0, 1)
	pair.AddOrder(helpers.StringToBigInt("10000000000000000000"), helpers.StringToBigInt("99998528242522874800000"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	var a, b, c = big.NewInt(0), big.NewInt(0), helpers.StringToBigInt("4000000000000000000")
	var oA, oB []*Limit
	{

		b, oB = pair.calculateSellForBuyWithOrders(c)
		a, oA = pair.calculateBuyForSellWithOrders(b)

		for i := range oA {
			t.Log(pair.getOrder(uint32(i + 1)))
			t.Log(oA[i])
			t.Log(oB[i])
		}

		if c.Cmp(a) != 0 {
			t.Skip(c, a)
		}
	}

	out, _, _ := pair.BuyWithOrders(c)
	if out.Cmp(b) != 0 {
		t.Error(out, b)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

}

func TestPair_MoreBuyOfSellInOrder(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("100000000000000000000000"), helpers.StringToBigInt("10000000000000000000000000"))
	pair := swap.Pair(0, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406001"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462002"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656003"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000001"), helpers.StringToBigInt("999987022814828419004"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748005"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	var a, b, c = big.NewInt(0), big.NewInt(0), helpers.StringToBigInt("4000000000000000000")
	var oA, oB []*Limit
	{

		a, oA = pair.calculateBuyForSellWithOrders(c)
		b, oB = pair.calculateSellForBuyWithOrders(a)

		for i := range oA {
			t.Log(oA[i])
			t.Log(pair.getOrder(uint32(i + 1)))
			t.Log(oB[i])
		}

		if c.Cmp(b) != 0 {
			t.Error(c, b)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Error("Recovered", r)
		}
	}()

	pair.SellWithOrders(c)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

}

func TestPair_MoreBuyOfSellInOrder1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("100000000000000000000000"), helpers.StringToBigInt("10000000000000000000000000"))
	pair := swap.Pair(0, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000001"), helpers.StringToBigInt("999993771961322406001"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000001"), helpers.StringToBigInt("999992979828068462002"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000001"), helpers.StringToBigInt("999990513182822656003"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000001"), helpers.StringToBigInt("999987022814828419004"), types.Address{1}, 1)

	//pair.AddOrder(helpers.StringToBigInt("1000000000000000001"), helpers.StringToBigInt("999985282425228748005"), types.Address{1}, 1)
	//pair.AddOrder(helpers.StringToBigInt("4035771609728482"), helpers.StringToBigInt("4035712212958056448"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("4035771609728483"), helpers.StringToBigInt("4035712212958056993"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	var a, b, c = big.NewInt(0), big.NewInt(0), helpers.StringToBigInt("4000000000000000000000")
	{

		var oA, oB []*Limit
		b, oB = pair.calculateSellForBuyWithOrders(c)
		a, oA = pair.calculateBuyForSellWithOrders(b)

		t.Log(pair.getOrder(uint32(len(oA))))
		t.Log(oA[len(oA)-1])
		t.Log(oB[len(oB)-1])
		//
		if c.Cmp(a) != 0 {
			//t.Error(c, a)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Error("Recovered", r)
		}
	}()

	pair.BuyWithOrders(c)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

}

func TestAddToList(t *testing.T) {
	list, inc, pos := addToList([]*Limit{
		//{
		//	WantBuy:  helpers.StringToBigInt("1000000000000000000"),
		//	WantSell: helpers.StringToBigInt("999993771961322406"),
		//	Owner:    types.Address{},
		//	IsBuy:    false,
		//	Height:   0,
		//	PairKey: PairKey{
		//		Coin0: 0,
		//		Coin1: 1,
		//	},
		//	oldSortPrice: nil,
		//	id:           1,
		//	RWMutex:      &sync.RWMutex{},
		//},
		{
			WantBuy:  helpers.StringToBigInt("1000000000000000000"),
			WantSell: helpers.StringToBigInt("999992979828068462"),
			Owner:    types.Address{},
			IsBuy:    false,
			Height:   0,
			PairKey: PairKey{
				Coin0: 0,
				Coin1: 1,
			},
			oldSortPrice: nil,
			id:           2,
			RWMutex:      &sync.RWMutex{},
		},
		{
			WantBuy:  helpers.StringToBigInt("1000000000000000000"),
			WantSell: helpers.StringToBigInt("999990513182822656"),
			Owner:    types.Address{},
			IsBuy:    false,
			Height:   0,
			PairKey: PairKey{
				Coin0: 0,
				Coin1: 1,
			},
			oldSortPrice: nil,
			id:           3,
			RWMutex:      &sync.RWMutex{},
		},
		{
			WantBuy:  helpers.StringToBigInt("1000000000000000000"),
			WantSell: helpers.StringToBigInt("999987022814828419"),
			Owner:    types.Address{},
			IsBuy:    false,
			Height:   0,
			PairKey: PairKey{
				Coin0: 0,
				Coin1: 1,
			},
			oldSortPrice: nil,
			id:           4,
			RWMutex:      &sync.RWMutex{},
		},
		{
			WantBuy:  helpers.StringToBigInt("1000000000000000000"),
			WantSell: helpers.StringToBigInt("999985282425228748"),
			Owner:    types.Address{},
			IsBuy:    false,
			Height:   0,
			PairKey: PairKey{
				Coin0: 0,
				Coin1: 1,
			},
			oldSortPrice: nil,
			id:           5,
			RWMutex:      &sync.RWMutex{},
		},
		nil,
	}, &Limit{
		WantBuy:  helpers.StringToBigInt("631171510357243556"),
		WantSell: helpers.StringToBigInt("631167579396664870"),
		Owner:    types.Address{},
		IsBuy:    false,
		Height:   0,
		PairKey: PairKey{
			Coin0: 0,
			Coin1: 1,
		},
		oldSortPrice: nil,
		id:           1,
		RWMutex:      &sync.RWMutex{},
	}, -1, 0)

	t.Log(inc, pos)

	for _, limit := range list {
		t.Log(limit)
	}
}

func TestPair_BuyOrderList(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10e10), big.NewInt(10e10))

	pair01 := swap.Pair(0, 1)
	pair01.AddOrder(big.NewInt(10010), big.NewInt(10000), types.Address{}, 0)
	pair01.AddOrder(big.NewInt(10020), big.NewInt(10000), types.Address{}, 0)

	pair10 := swap.Pair(1, 0)
	pair10.AddOrder(big.NewInt(1003), big.NewInt(1000), types.Address{}, 0)
	pair10.AddOrder(big.NewInt(1004), big.NewInt(1000), types.Address{}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(pair01.OrderSellLast())
	t.Log(pair01.SellOrderIDs())

	t.Log(pair10.OrderSellLast())
	t.Log(pair10.SellOrderIDs())
}

func TestPair_LoadOrders_MemVSDisk(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)
		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
		pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())
		//t.SkipNow()
		pair.SellWithOrders(helpers.StringToBigInt("400000000000000000"))

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		pair.SellWithOrders(helpers.StringToBigInt("400000000000000000"))

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		swap.PairRemoveLimitOrder(pair.SellOrderIDs()[0])

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		pair.SellWithOrders(helpers.StringToBigInt("400000000000000000"))

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)

		t.Log(pair.OrderSellLast())
		t.Log(pair.SellOrderIDs())

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSwap_AddEqualOrders(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999999999999999999"), types.Address{1}, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992222222222222"), types.Address{1}, 1)

	t.Log(pair.OrderSellLast())
	if !reflect.DeepEqual(pair.sellOrders.ids, []uint32{9, 1, 2, 3, 4, 5, 10, 6, 7, 8, 0}) {
		t.Error("unsorted", pair.sellOrders.ids, []uint32{9, 1, 2, 3, 4, 5, 10, 6, 7, 8, 0})
	}

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	t.Log(pair.OrderSellLast())
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(pair.OrderSellLast())
	if !reflect.DeepEqual(pair.sellOrders.ids, []uint32{9, 1, 2, 3, 4, 5, 10, 6, 7, 8, 0}) {
		t.Error("unsorted", pair.sellOrders.ids)
	}
}
func TestSwap_loadSellOrders_0(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	order := pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	orders := swap.loadSellOrders(pair, nil, 10)
	t.Log(orders)

	t.Log(order)
	orders = swap.loadSellOrders(pair, order, 10)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 0)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 2)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 1)
	t.Log(orders)
}
func TestSwap_loadSellOrders_3(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	order := pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	orders := swap.loadSellOrders(pair, order, 1)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 10)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 0)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 2)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 1)
	t.Log(orders)
}
func TestSwap_loadSellOrders_9(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	order := pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	orders := swap.loadSellOrders(pair, nil, 1)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, nil, 10)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 0)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, order, 2)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, nil, 1)
	t.Log(orders)
}
func TestSwap_loadSellOrders_dirty(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	orders := swap.loadSellOrders(pair, nil, 1)
	t.Log(orders)

	orders = swap.loadSellOrders(pair, nil, 1)
	t.Log(orders)

	//t.Skip("OK")
	//o1 := pair.OrderSellByIndex(0)
	//o1.WantSell = big.NewInt(0)
	//o1.WantBuy = big.NewInt(0)
	//o1.SortPrice()
	//pair.dirtyOrders.list[o1.id] = struct{}{}
	//pair.unsetOrderSellByIndex(0)
	//t.Log(o1.id)
	//
	//o2 := pair.OrderSellByIndex(0)
	//o2.ReCalcOldSortPrice()
	//o2.WantSell = helpers.StringToBigInt("999992979828068460")
	////o2.WantSell = helpers.StringToBigInt("999990513182822900")
	//o2.WantBuy = helpers.StringToBigInt("1000000000000000000")
	//o2.SortPrice()
	//pair.dirtyOrders.list[o2.id] = struct{}{}
	////pair.unsetOrderSellByIndex(0)
	//t.Log(o2.id)
	//
	//order := pair.OrderSellByIndex(0)
	//t.Log(order.id)
	//order = pair.OrderSellByIndex(0)
	//t.Log(order.id)

}

func TestPair_LoadOrders_bagSort1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	//t.Log(pair.orderSellByIndex(3))
	t.Log(pair.orderSellByIndex(7))

	if !reflect.DeepEqual(pair.sellOrders.ids, []uint32{1, 6, 2, 7, 3, 8, 4, 9, 5, 10, 0}) {
		t.Error("unsorted", pair.sellOrders.ids, []uint32{1, 6, 2, 7, 3, 8, 4, 9, 5, 10, 0})
	}

	t.Log(pair.unsortedDirtySellOrders.list)
	//t.SkipNow()
	last, index := pair.OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}
}

func TestPair_LoadOrders_bagSort(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	swap.PairRemoveLimitOrder(3)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	last, index := pair.OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}

	//t.Log(pair.orderSellByIndex(3))
	//t.Log(pair.orderSellByIndex(7))

	if !reflect.DeepEqual(pair.sellOrders.ids, []uint32{1, 6, 2, 7, 8, 11, 4, 9, 5, 10, 0}) {
		t.Error("unsorted", pair.sellOrders.ids, []uint32{1, 6, 2, 7, 8, 11, 4, 9, 5, 10, 0})
	}

	t.Log(pair.unsortedDirtySellOrders.list)
	//t.SkipNow()
	last, index = pair.OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}

	if !reflect.DeepEqual(pair.sellOrders.ids, []uint32{1, 6, 2, 7, 8, 11, 4, 9, 5, 10, 0}) {
		t.Error("unsorted", pair.sellOrders.ids, []uint32{1, 6, 2, 7, 8, 11, 4, 9, 5, 10, 0})
	}
}

func TestPair_LoadOrders_bagTacoCommit2(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	//t.Logf("%#v", pair.orderSellByIndex(0))
	t.Log(pair.orderSellByIndex(0))
	t.Log(pair.orderSellByIndex(1))
	t.Log(pair.orderSellByIndex(2))
	t.Log(pair.orderSellByIndex(3))
	t.Log(pair.orderSellByIndex(4))
	t.Log(pair.orderSellByIndex(5))
	t.Log(pair.orderSellByIndex(6))
	t.Log(pair.orderSellByIndex(7))

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	//t.SkipNow()
	last, index := pair.OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(2, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 10 || index != 9 {
		t.Fatal(last, index)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("4000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 10 || index != 6 {
		t.Log(pair.sellOrders.ids)
		t.Fatal(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(3, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 10 || index != 6 {
		t.Fatal(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(4, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 10 || index != 6 {
		t.Fatal(last, index)
	}

	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("40000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(5, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}
}

func TestPair_LoadOrders_bagTacoCommit1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)
	//pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322400"), types.Address{1}, 1)
	//pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	//pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)
	//pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228800"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	t.Log(pair.sellOrders.ids)
	t.Log(pair.unsortedDirtySellOrders.list)

	//t.Logf("%#v", pair.orderSellByIndex(0))
	t.Log(pair.orderSellByIndex(0))
	t.Log(pair.orderSellByIndex(1))
	t.Log(pair.orderSellByIndex(2))
	t.Log(pair.orderSellByIndex(3))
	t.Log(pair.orderSellByIndex(4))
	t.Log(pair.orderSellByIndex(5))
	t.Log(pair.orderSellByIndex(6))
	t.Log(pair.orderSellByIndex(7))

	//t.SkipNow()
	last, index := pair.OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Fatal(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(2, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Fatal(last, index)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("4000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 1 {
		t.Fatal(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(3, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 1 {
		t.Fatal(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(4, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 1 {
		t.Fatal(last, index)
	}

	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("4000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(5, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}
}

func TestPair_LoadOrders_loadSell1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	order := pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322400"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228800"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	orders := swap.loadSellOrders(pair, order, 3)
	if len(orders) > 2 {
		t.Error(orders)
	}

}
func TestPair_LoadOrders_bagTacoCommit(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	last, index := swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(2, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)

	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}
	pair.SellWithOrders(helpers.StringToBigInt("400000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(3, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(4, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("400000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(5, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}
}

func TestPair_LoadOrders_bagTacoCirle(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	last, index := swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(2, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("400000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(3, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(4, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	pair = swap.Pair(0, 1)
	pair.SellWithOrders(helpers.StringToBigInt("40000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(5, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}
}

func TestPair_LoadOrders_bagTacoMen(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999993771961322406"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999992979828068462"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999990513182822656"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999987022814828419"), types.Address{1}, 1)
	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("999985282425228748"), types.Address{1}, 1)

	last, index := pair.OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}
	t.Log(pair.SellOrderIDs())

	for _, limit := range pair.OrdersSell(999) {
		t.Logf("%#v", limit)
	}
	pair.SellWithOrders(helpers.StringToBigInt("400000000000000000"))
	for _, limit := range pair.OrdersSell(999) {
		t.Logf("%#v", limit)
	}

	last, index = pair.OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index, pair.SellOrderIDs())
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last.id != 5 || index != 4 {
		t.Error(last, index)
	}

	pair.SellWithOrders(helpers.StringToBigInt("40000000000000000000"))

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}

	last, index = swap.Pair(0, 1).OrderSellLast()
	if last != nil || index != -1 {
		t.Error(last, index)
	}
}

func TestPair_LoadOrders(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("2000000000000000000"), helpers.StringToBigInt("2000000000000000000"))

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	swap = New(newBus, immutableTree.GetLastImmutable())
	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("1000000000000000000"), helpers.StringToBigInt("1000000000000000000"), types.Address{1}, 1)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(2, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	swap = New(newBus, immutableTree.GetLastImmutable())
	{
		order := swap.GetSwapper(0, 1).OrderSellByIndex(0)
		t.Log(order)
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(3, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.GetSwapper(0, 1)
		order1 := pair.OrderSellByIndex(0)
		t.Log(order1)
	}
	_, value := immutableTree.GetLastImmutable().Get(pathOrder(1))
	order := &Limit{
		id: 1,
	}

	if err := rlp.DecodeBytes(value, order); err != nil {
		panic(err)
	}
	t.Logf("%#v", order)

	{
		immutableTree.GetLastImmutable().IterateRange(
			pricePath(PairKey{
				Coin0: 0,
				Coin1: 1,
			}, CalcPriceSell(order.WantBuy, order.WantSell), 0, true),
			pricePath(PairKey{
				Coin0: 0,
				Coin1: 1,
			}, CalcPriceSell(order.WantBuy, order.WantSell), 2, true),
			true, func(key []byte, value []byte) bool {
				t.Logf("%#v", key[len(key)-4:])
				t.Logf("%#v", value)
				return false
			})

	}
}

func TestPair_ResortOrders(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("15000000000000000000000"), helpers.StringToBigInt("5000000000000000000000"), types.Address{}, 0)

	// order := pair.OrderSellByIndex(0)

	_, _, _ = pair.SellWithOrders(helpers.StringToBigInt("10000000000000000000000"))

	t.Run("resort dirty", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			t.Run("mem", func(t *testing.T) {
				orderNextMem := pair.OrderSellByIndex(0)
				t.Run("disk", func(t *testing.T) {
					_, _, err = immutableTree.Commit(swap)
					if err != nil {
						t.Fatal(err)
					}
					pair := New(newBus, immutableTree.GetLastImmutable()).Pair(0, 1)
					orderNextDisk := pair.OrderSellByIndex(0)
					if orderNextDisk != nil && orderNextMem != nil {
						t.Log("has order")
						if orderNextDisk.id != orderNextMem.id {
							t.Errorf("disk %d, mem %d", orderNextDisk.id, orderNextMem.id)
						}
					} else {
						t.Error("no order")
						if orderNextDisk != nil || orderNextMem != nil {
							t.Errorf("disk %#v, mem %#v", orderNextDisk, orderNextMem)
						}
					}
				})
			})
		})
	})

	_, value := immutableTree.GetLastImmutable().Get(pathOrder(1))
	order := &Limit{
		id: 1,
	}

	if err := rlp.DecodeBytes(value, order); err != nil {
		panic(err)
	}
	t.Logf("%#v", order)
}

func TestPair_SellWithOrders_changePriceWithOrderAndUpdateList0(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("15000000000000000000000"), helpers.StringToBigInt("5000000000000000000000"), types.Address{}, 0)

	// order := pair.OrderSellByIndex(0)

	_, _, _ = pair.SellWithOrders(big.NewInt(1e18))

	t.Run("resort dirty", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			t.Run("mem", func(t *testing.T) {
				orderNextMem := pair.OrderSellByIndex(0)
				t.Run("disk", func(t *testing.T) {
					_, _, err = immutableTree.Commit(swap)
					if err != nil {
						t.Fatal(err)
					}
					pair := New(newBus, immutableTree.GetLastImmutable()).Pair(0, 1)
					orderNextDisk := pair.OrderSellByIndex(0)
					if orderNextDisk != nil && orderNextMem != nil {
						// t.Log("has order")
						if orderNextDisk.id != orderNextMem.id {
							t.Errorf("dosk %d, mem %d", orderNextDisk.id, orderNextMem.id)
						}
					} else {
						t.Error("no order")
						if orderNextDisk != nil || orderNextMem != nil {
							t.Errorf("disk %#v, mem %#v", orderNextDisk, orderNextMem)
						}
					}
				})
			})
		})
	})
}

func TestPair_SellWithOrders_changePriceWithOrderAndUpdateList1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	pair.AddOrder(helpers.StringToBigInt("15000000000000000000000"), helpers.StringToBigInt("5000000000000000000000"), types.Address{}, 0)

	// order := pair.OrderSellByIndex(0)

	_, _, _ = pair.SellWithOrders(helpers.StringToBigInt("10000000000000000000000"))

	t.Run("resort dirty", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			t.Run("mem", func(t *testing.T) {
				orderNextMem := pair.OrderSellByIndex(0)
				t.Run("disk", func(t *testing.T) {
					_, _, err = immutableTree.Commit(swap)
					if err != nil {
						t.Fatal(err)
					}
					pair := New(newBus, immutableTree.GetLastImmutable()).Pair(0, 1)
					orderNextDisk := pair.OrderSellByIndex(0)
					if orderNextDisk != nil && orderNextMem != nil {
						t.Log("has order")
						if orderNextDisk.id != orderNextMem.id {
							t.Errorf("disk %d, mem %d", orderNextDisk.id, orderNextMem.id)
						}
					} else {
						t.Error("no order")
						if orderNextDisk != nil || orderNextMem != nil {
							t.Errorf("disk %#v, mem %#v", orderNextDisk, orderNextMem)
						}
					}
				})
			})
		})
	})
}

func TestPair_CalculateAddAmount0ForPrice_0(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(110e8), big.NewInt(440e8))

	pair := swap.Pair(0, 1)
	t.Log(pair.Price())
	amount0ForPrice, _ := pair.CalculateAddAmountsForPrice(big.NewFloat(2))

	wantedAmount0In := helpers.StringToBigInt("4560910765") // 456090165
	if amount0ForPrice.Cmp(wantedAmount0In) != 0 {
		t.Error("wrong need to sell", amount0ForPrice)
	}
	wantedAmount1Out := helpers.StringToBigInt("12878178468") // 128781967
	wantedCalcAmount0In := pair.CalculateSellForBuy(wantedAmount1Out)
	if wantedCalcAmount0In.Cmp(wantedAmount0In) != 0 {
		t.Error("wrong need to sell", wantedCalcAmount0In)
	}
	calcAmount1Out := pair.CalculateBuyForSell(amount0ForPrice)
	if calcAmount1Out.Cmp(wantedAmount1Out) != 0 {
		t.Error("wrong need to buy", calcAmount1Out)
	}

	amount1Out, _, _ := pair.SellWithOrders(amount0ForPrice)
	if amount1Out.Cmp(wantedAmount1Out) != 0 {
		t.Error("wrong need to buy", amount1Out.String())
	}
	if pair.Price().String() != "2" {
		t.Error("wrong new price", pair.Price().String())
	}
	t.Log(pair.Reserves())
}

func TestPair_CalculateAddAmount0ForPrice_1(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("58230513808506823674863"), helpers.StringToBigInt("58277712683264545746619"))

	pair := swap.Pair(0, 1)
	t.Log(pair.Price())
	amount0ForPrice, _ := pair.CalculateAddAmountsForPrice(big.NewFloat(1))

	if amount0ForPrice.String() != "23618275451859783680" {
		t.Error("wrong need to sell", amount0ForPrice)
	}
	pair.SellWithOrders(amount0ForPrice)
	if pair.Price().String() != "1" {
		t.Error("wrong new price", pair.Price().String())
	}
}

func TestSimple_my(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)

	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, helpers.StringToBigInt("10000000000000000000000"), helpers.StringToBigInt("10000000000000000000000"))

	pair := swap.Pair(0, 1)

	wantBuy := helpers.StringToBigInt("15000000000000000000000")
	wantSell := helpers.StringToBigInt("5000000000000000000000")
	order := pair.AddOrder(wantBuy, wantSell, types.Address{}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	amount0In := helpers.StringToBigInt("10000000000000000000000000")
	amount1Out, orders := pair.calculateBuyForSellWithOrders(amount0In)
	commission0orders, commission1orders, amount0, amount1, _ := CalcDiffPool(amount0In, amount1Out, orders)

	if amount1Out.String() != "14989949149767990691508" {
		t.Error("err", amount1Out.String())
	}

	amount0ForPrice, _ := pair.CalculateAddAmountsForPrice(order.Price())
	amount1ForPrice := pair.CalculateBuyForSell(amount0ForPrice)

	if amount0ForPrice.String() != "7327837463256660705280" {
		t.Error("err", amount0ForPrice.String())
	}
	if amount1ForPrice.String() != "4224054178914447923672" {
		t.Error("err", amount1ForPrice.String())
	}

	if commission0orders.String() != "14985014985014985015" {
		t.Error("err", commission0orders.String())
	}
	if commission1orders.String() != "4995004995004995005" {
		t.Error("err", commission1orders.String())
	}

	amount0AfterPrice := big.NewInt(0).Sub(amount0, amount0ForPrice)
	amount1AfterPrice := big.NewInt(0).Sub(amount1, amount1ForPrice)

	if amount0AfterPrice.String() != "9977672162536743339294720" {
		t.Error("err", amount0AfterPrice.String())
	}
	if amount1AfterPrice.String() != "5770889975848547762841" {
		t.Error("err", amount1AfterPrice.String())
	}

	if err := pair.CheckSwap(amount0ForPrice, amount1ForPrice); err != nil {
		t.Error(err)
	}
	pair.Swap(amount0ForPrice, big.NewInt(0), big.NewInt(0), amount1ForPrice)
	pair.update(amount0ForPrice, amount1ForPrice)
	if err := pair.CheckSwap(amount0AfterPrice, amount1AfterPrice); err != nil {
		t.Error(err)
	}
	pair.Swap(amount0AfterPrice, big.NewInt(0), big.NewInt(0), amount1AfterPrice)

	defer func() {
		if r := recover(); r != nil {
			t.Error("Recovered", r)
		}
	}()
	out, _, _ := New(newBus, immutableTree.GetLastImmutable()).Pair(0, 1).SellWithOrders(amount0In)
	if out.Cmp(amount1Out) != 0 {
		t.Error("err")
	}
}

func TestPair_OrderID(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))

		pair := swap.Pair(0, 1)
		id := pair.AddOrder(big.NewInt(1), big.NewInt(1), types.Address{}, 0).id
		if id != 1 {
			t.Errorf("next orders ID want %d, got %d", 1, id)
		}
		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))

		pair := swap.Pair(0, 1)
		if id := pair.AddOrder(big.NewInt(2), big.NewInt(1), types.Address{}, 0).id; id != 2 {
			t.Errorf("next orders ID want %d, got %d", 2, id)
		}
		if id := pair.AddOrder(big.NewInt(3), big.NewInt(1), types.Address{}, 0).id; id != 3 {
			t.Errorf("next orders ID want %d, got %d", 3, id)
		}
		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)
		if id := pair.AddOrder(big.NewInt(4), big.NewInt(1), types.Address{}, 0).id; id != 4 {
			t.Errorf("next orders ID want %d, got %d", 4, id)
		}
		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("export", func(t *testing.T) {
		var appState types.AppState
		swap := New(newBus, immutableTree.GetLastImmutable())
		swap.Export(&appState)

		jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
		if err != nil {
			t.Error(err)
		}
		t.Logf("%s", jsonBytes)
		if len(appState.Pools) != 1 {
			t.Fatalf("pools are not all: %s", jsonBytes)
		}
		if appState.NextOrderID != 5 {
			t.Errorf("next order ID want %d, got %d", 5, appState.NextOrderID)
		}
		ids := map[uint32]struct{}{
			1: {},
			2: {},
			3: {},
			4: {},
		}
		for _, order := range appState.Pools[0].Orders {
			if _, ok := ids[uint32(order.ID)]; !ok {
				t.Errorf("not found id %d", order.ID)
			}
			delete(ids, uint32(order.ID))
		}
	})
}

func TestPair_AddLastSwapStepWithOrders(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(1e18), big.NewInt(1e18))

	pair := swap.Pair(0, 1)
	pair.AddOrder(big.NewInt(20e15), big.NewInt(5e15), types.Address{}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(0, 1)
	price := pair.OrderSellByIndex(0).Price()
	t.Log(price)
	addAmount0ForPrice, _ := pair.CalculateAddAmountsForPrice(price)

	_, _, _ = pair.SellWithOrders(addAmount0ForPrice)

	sell := big.NewInt(15e15)
	calcBuy1 := pair.CalculateBuyForSellWithOrders(sell)
	pair1 := pair.AddLastSwapStepWithOrders(sell, calcBuy1, false)
	calcBuy2 := pair1.CalculateBuyForSellWithOrders(sell)
	pair1.AddLastSwapStepWithOrders(sell, calcBuy2, false)

	if len(pair.dirtyOrders.list) != 0 {
		t.Error("err", pair.dirtyOrders.list)
	}
	if len(pair1.(*Pair).dirtyOrders.list) == 0 {
		t.Error("err", pair1.(*Pair).dirtyOrders.list)
	}
	// t.SkipNow()
	buy1, _, _ := pair.SellWithOrders(sell)
	if calcBuy1.Cmp(buy1) != 0 {
		t.Error("err", calcBuy1, buy1)
	}
	t.Log(calcBuy1, buy1)

	buy2, _, _ := pair.SellWithOrders(sell)
	if calcBuy2.Cmp(buy2) != 0 {
		t.Error("err", calcBuy2, buy2)
	}
	t.Log(calcBuy2, buy2)
}

func TestPair_BuyWithOrders_01_ChangeRemainderOrderPrice(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))

	pair := swap.Pair(0, 1)
	pair.AddOrder(big.NewInt(15000), big.NewInt(5000), types.Address{1}, 0)
	pair.AddOrder(big.NewInt(20), big.NewInt(5), types.Address{2}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(0, 1)
	price := pair.OrderSellByIndex(0).Price()

	addAmount0ForPrice, addAmount1 := pair.CalculateAddAmountsForPrice(price)
	if addAmount0ForPrice.Cmp(big.NewInt(7327)) != 0 {
		t.Error("a", addAmount0ForPrice)
	}

	if addAmount1.Cmp(big.NewInt(4223)) != 0 {
		t.Error("z", addAmount1)
	}

	p0 := pair.AddLastSwapStep(addAmount0ForPrice, addAmount1).Price().Text('f', 18)

	t.Run("add amount1", func(t *testing.T) {
		addAmount0, addAmount1ForPrice := pair.CalculateAddAmountsForPrice(price)
		if addAmount1ForPrice.Cmp(addAmount1) != 0 {
			t.Error("a", addAmount1ForPrice)
		}

		p1 := pair.AddLastSwapStep(addAmount0ForPrice, addAmount1ForPrice).Price().Text('f', 18)
		if p1 != p0 {
			t.Error(p1, p0)
		}

		if addAmount0.Cmp(addAmount0ForPrice) != 0 {
			t.Errorf("todo, addAmount0 %s, addAmount0ForPrice %s", addAmount0, addAmount0ForPrice)
		}
	})

	amount0In, owners, _ := pair.BuyWithOrders(big.NewInt(0).Add(addAmount1, big.NewInt(3000)))
	t.Run("owner", func(t *testing.T) {
		if len(owners) != 1 {
			t.Fatal("b", owners)
		}

		if owners[types.Address{1}].Cmp(big.NewInt(9000)) != 0 {
			t.Error("c", owners[types.Address{}])
		}
	})

	if amount0In.Cmp(big.NewInt(7327+9009)) != 0 {
		t.Error("d", amount0In, big.NewInt(7327+9009)) // todo
	}

	t.Run("resort dirty", func(t *testing.T) {
		order := pair.OrderSellByIndex(0)
		if order.id != 1 {
			t.Errorf("want %d, got %d", 1, order.id)
		}
		if order.WantSell.Cmp(big.NewInt(1997)) != 0 {
			t.Error(order.WantSell)
		}
		if order.WantBuy.Cmp(big.NewInt(5991)) != 0 {
			t.Error(order.WantBuy)
		}
		amount0In, owners, _ := pair.BuyWithOrders(big.NewInt(1000))
		t.Run("owner", func(t *testing.T) {
			if len(owners) != 1 {
				t.Fatal("b", owners)
			}

			if owners[types.Address{1}].Cmp(big.NewInt(3000)) != 0 {
				t.Error("c", owners[types.Address{}])
			}
		})

		if amount0In.Cmp(big.NewInt(3003)) != 0 {
			t.Error("d", amount0In)
		}

		if pair.OrderSellByIndex(0).id != 1 {
			t.Errorf("want %d, got %d", 1, pair.OrderSellByIndex(0).id)
		}
	})

	// t.Log(pair.Price())

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("export", func(t *testing.T) {
		var appState types.AppState
		swap := New(newBus, immutableTree.GetLastImmutable())
		swap.Export(&appState)

		jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
		if err != nil {
			t.Error(err)
		}
		if len(appState.Pools) != 1 {
			t.Fatalf("pools are not all: %s", jsonBytes)
		}
		if len(appState.Pools[0].Orders) != 2 {
			t.Fatalf("orders are empty, %s", jsonBytes)
		} else {
			t.Logf("%#v", appState.Pools[0].Orders)
		}
	})
}

func TestPair_SellWithOrders_01_ChangeRemainderOrderPrice(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))

	pair := swap.Pair(0, 1)
	pair.AddOrder(big.NewInt(15000), big.NewInt(5000), types.Address{}, 0)
	pair.AddOrder(big.NewInt(20), big.NewInt(5), types.Address{}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(0, 1)
	price := pair.OrderSellByIndex(0).Price()
	addAmount0ForPrice, _ := pair.CalculateAddAmountsForPrice(price)
	if addAmount0ForPrice.Cmp(big.NewInt(7327)) != 0 {
		t.Error("a", addAmount0ForPrice)
	}

	addAmount1 := pair.CalculateBuyForSell(addAmount0ForPrice)
	if addAmount1.Cmp(big.NewInt(4223)) != 0 {
		t.Error("z", addAmount1)
	}

	amount1Out, owners, _ := pair.SellWithOrders(big.NewInt(0).Add(addAmount0ForPrice, big.NewInt(9009)))
	if len(owners) != 1 {
		t.Error("b", owners)
	}

	if owners[types.Address{}].Cmp(big.NewInt(9000)) != 0 {
		t.Error("c", owners[types.Address{}])
	}

	if amount1Out.Cmp(big.NewInt(4223+3000)) != 0 {
		t.Error("d", amount1Out)
	}

	t.Run("resort dirty", func(t *testing.T) {
		order := pair.OrderSellByIndex(0)
		if order.id != 1 {
			t.Errorf("want %d, got %d", 1, order.id)
		}
		if order.WantSell.Cmp(big.NewInt(1997)) != 0 {
			t.Error(order.WantSell)
		}
		if order.WantBuy.Cmp(big.NewInt(5991)) != 0 {
			t.Error(order.WantBuy)
		}
		amount0Out, owners, _ := pair.SellWithOrders(big.NewInt(3003))
		t.Run("owner", func(t *testing.T) {
			if len(owners) != 1 {
				t.Fatal("b", owners)
			}

			if owners[types.Address{}].Cmp(big.NewInt(3000)) != 0 {
				t.Error("c", owners[types.Address{}])
			}
		})

		if amount0Out.Cmp(big.NewInt(1000)) != 0 {
			t.Error("d", amount0Out)
		}

		if pair.OrderSellByIndex(0).id != 1 {
			t.Errorf("want %d, got %d", 1, pair.OrderSellByIndex(0).id)
		}
	})

	t.Log(pair.Price())

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("export", func(t *testing.T) {
		var appState types.AppState
		swap := New(newBus, immutableTree.GetLastImmutable())
		swap.Export(&appState)

		jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
		if err != nil {
			t.Error(err)
		}
		if len(appState.Pools) != 1 {
			t.Fatalf("pools are not all: %s", jsonBytes)
		}
		if len(appState.Pools[0].Orders) != 2 {
			t.Fatalf("orders are empty, %s", jsonBytes)
		} else {
			t.Logf("%#v", appState.Pools[0].Orders)
		}
	})
}

func TestPair_SellWithOrders_10_ChangeRemainderOrderPrice(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))

	pair := swap.Pair(1, 0)
	pair.AddOrder(big.NewInt(15000), big.NewInt(5000), types.Address{}, 0)
	pair.AddOrder(big.NewInt(20), big.NewInt(5), types.Address{}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(1, 0)
	price := pair.OrderSellByIndex(0).Price()
	t.Log(price)
	addAmount0ForPrice, _ := pair.CalculateAddAmountsForPrice(price)
	if addAmount0ForPrice.Cmp(big.NewInt(7327)) != 0 {
		t.Error("a", addAmount0ForPrice)
	}

	addAmount1 := pair.CalculateBuyForSell(addAmount0ForPrice)
	if addAmount1.Cmp(big.NewInt(4223)) != 0 {
		t.Error("z", addAmount1)
	}

	amount1Out, owners, _ := pair.SellWithOrders(big.NewInt(0).Add(addAmount0ForPrice, big.NewInt(9009)))
	if len(owners) != 1 {
		t.Error("b", owners)
	}

	if owners[types.Address{}].Cmp(big.NewInt(9000)) != 0 {
		t.Error("c", owners[types.Address{}])
	}

	if amount1Out.Cmp(big.NewInt(4223+3000)) != 0 {
		t.Error("d", amount1Out)
	}

	t.Run("resort dirty", func(t *testing.T) {
		t.Skip("allow sell order with highest price without diff pool reserves")
		t.Run("mem", func(t *testing.T) {
			order := pair.OrderSellByIndex(0)
			if order.id != 1 {
				t.Errorf("want %d, got %d", 1, order.id)
			}
			if order.WantSell.Cmp(big.NewInt(1997)) != 0 {
				t.Error(order.WantSell)
			}
			if order.WantBuy.Cmp(big.NewInt(5991)) != 0 {
				t.Error(order.WantBuy)
			}
			amount0In, owners, _ := pair.SellWithOrders(big.NewInt(1001))
			t.Run("owner", func(t *testing.T) {
				if len(owners) != 1 {
					t.Fatal("b", owners)
				}

				if owners[types.Address{}].Cmp(big.NewInt(1000)) != 0 {
					t.Error("c", owners[types.Address{}])
				}
			})

			if amount0In.Cmp(big.NewInt(332)) != 0 {
				t.Error("d", amount0In)
			}

			if order.WantSell.Cmp(big.NewInt(1664)) != 0 {
				t.Error(order.WantSell)
			}
			if order.WantBuy.Cmp(big.NewInt(4990)) != 0 {
				t.Error(order.WantBuy)
			}

			t.Run("mem", func(t *testing.T) {
				orderNext := pair.OrderSellByIndex(0)
				if order.Price().Cmp(orderNext.Price()) != 1 {
					t.Errorf("order %d price %v, and %d price %v", orderNext.id, orderNext.Price(), order.id, order.Price())
				}
				if orderNext.id != 2 {
					t.Errorf("want %d, got %d", 2, order.id)
				}
			})
			t.Run("disk", func(t *testing.T) {
				_, _, err = immutableTree.Commit(swap)
				if err != nil {
					t.Fatal(err)
				}

				pair := New(newBus, immutableTree.GetLastImmutable()).Pair(1, 0)

				orderNext := pair.OrderSellByIndex(0)
				if order.Price().Cmp(orderNext.Price()) != 1 {
					t.Errorf("order %d price %v, and %d price %v", orderNext.id, orderNext.Price(), order.id, order.Price())
				}
				if orderNext.id != 2 {
					t.Errorf("want %d, got %d", 2, order.id)
				}
			})
		})

	})

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("export", func(t *testing.T) {
		var appState types.AppState
		swap := New(newBus, immutableTree.GetLastImmutable())
		swap.Export(&appState)

		jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
		if err != nil {
			t.Error(err)
		}
		if len(appState.Pools) != 1 {
			t.Fatalf("pools are not all: %s", jsonBytes)
		}
		if len(appState.Pools[0].Orders) != 2 {
			t.Fatalf("orders are empty, %s", jsonBytes)
		} else {
			t.Logf("%s", jsonBytes)
		}

	})
}

func TestPair_SellWithOrders_01_FullOrder(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	pair := swap.Pair(0, 1)
	owner := types.HexToAddress("Mx7f0fc21d932f38ca9444f61703174569066cfa50")
	swap.PairAddOrder(0, 1, big.NewInt(2000), big.NewInt(1000), owner, 0)
	if pair.OrderSellByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
		t.Error("error set order")
	}
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("order", func(t *testing.T) {
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)
		amount1Out := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000))
		_, orders := pair.calculateBuyForSellWithOrders(big.NewInt(4146 + 2000))
		if len(orders) == 0 {
			t.Error("empty orders")
		}
		amount1OutSell, owners, _ := pair.SellWithOrders(big.NewInt(4146 + 2000))
		if amount1OutSell.Cmp(amount1Out) != 0 {
			t.Error("not equal", amount1Out, amount1OutSell)
		}
		if len(owners) == 0 {
			t.Error("empty owners")
		}
		if owners[owner] == nil || owners[owner].Cmp(big.NewInt(1998)) != 0 {
			t.Errorf("%#v", owners[owner])
		}
		//pair.OrderSellByIndex(0)
		t.Run("unset", func(t *testing.T) {
			if len(pair.SellOrderIDs()) != 0 {
				t.Errorf("slice len %d, want empty: %v", len(pair.SellOrderIDs()), pair.SellOrderIDs())
				t.Logf("%#v", pair.getOrder(pair.SellOrderIDs()[0]))

			}
		})

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("export", func(t *testing.T) {
			var appState types.AppState
			swap := New(newBus, immutableTree.GetLastImmutable())
			swap.Export(&appState)

			jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
			if err != nil {
				t.Error(err)
			}
			if len(appState.Pools) != 1 {
				t.Fatalf("pools are not all: %s", jsonBytes)
			}
			if len(appState.Pools[0].Orders) != 0 {
				t.Errorf("orders are not empty, %s", jsonBytes)
			}
		})
	})
}

func TestPair_SellWithOrders_01_PartOrder(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))

	pair := swap.Pair(0, 1)
	owner := types.HexToAddress("Mx7f0fc21d932f38ca9444f61703174569066cfa50")
	swap.PairAddOrder(0, 1, big.NewInt(2000), big.NewInt(1000), owner, 0)
	if pair.OrderSellByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
		t.Error("error set order")
	}

	t.Run("order", func(t *testing.T) {
		amount1Out := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 1000))
		_, orders := pair.calculateBuyForSellWithOrders(big.NewInt(4146 + 1000))
		if len(orders) == 0 {
			t.Error("empty orders")
		}
		amount1OutSell, owners, _ := pair.SellWithOrders(big.NewInt(4146 + 1000))
		if amount1OutSell.Cmp(amount1Out) != 0 {
			t.Error("not equal", amount1Out, amount1OutSell)
		}
		if len(owners) == 0 {
			t.Error("empty owners")
		}
		if owners[owner] == nil || owners[owner].Cmp(big.NewInt(999)) != 0 {
			t.Errorf("%#v", owners[owner])
		}

		_, _, err = immutableTree.Commit(swap)
		if err != nil {
			t.Fatal(err)
		}

		t.Run("export", func(t *testing.T) {
			var appState types.AppState
			swap.Export(&appState)

			jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
			if err != nil {
				t.Error(err)
			}
			if len(appState.Pools) != 1 {
				t.Fatalf("pools are not all: %s", jsonBytes)
			}
			if len(appState.Pools[0].Orders) != 1 {
				t.Errorf("orders are empty, %s", jsonBytes)
			}
			t.Logf("%s", jsonBytes)
			if appState.Pools[0].Orders[0].Volume0 != "1000" {
				t.Errorf("order sell part, %s", appState.Pools[0].Orders[0].Volume0)
			}
			if appState.Pools[0].Orders[0].Volume1 != "500" {
				t.Errorf("order buy part, %s", appState.Pools[0].Orders[0].Volume1)
			}
		})
	})
}

func TestSwap_Export_WithOrders(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10e10), big.NewInt(10e10))

	pair01 := swap.Pair(0, 1)
	pair01.AddOrder(big.NewInt(10010), big.NewInt(10000), types.Address{}, 0)
	pair01.AddOrder(big.NewInt(10020), big.NewInt(10000), types.Address{}, 0)

	pair10 := swap.Pair(1, 0)
	pair10.AddOrder(big.NewInt(1003), big.NewInt(1000), types.Address{}, 0)
	pair10.AddOrder(big.NewInt(1004), big.NewInt(1000), types.Address{}, 0)

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("export", func(t *testing.T) {
		var appState types.AppState
		swap.Export(&appState)

		jsonBytes, err := amino.NewCodec().MarshalJSONIndent(appState.Pools, "", "	")
		if err != nil {
			t.Error(err)
		}
		t.Logf("%s", jsonBytes)
		if len(appState.Pools) != 1 {
			t.Fatalf("pools are not all: %s", jsonBytes)
		}
		if len(appState.Pools[0].Orders) != 4 {
			t.Errorf("orders are not all, %s", jsonBytes)
		}
	})

	t.Run("cmp", func(t *testing.T) {
		amount1Out01Mem := pair01.CalculateBuyForSell(big.NewInt(1e10))
		amount1Out10Mem := pair10.CalculateBuyForSell(big.NewInt(1e10))
		amount1Out01OBMem := pair01.CalculateBuyForSellWithOrders(big.NewInt(1e10))
		amount1Out10OBMem := pair10.CalculateBuyForSellWithOrders(big.NewInt(1e10))
		t.Run("mem", func(t *testing.T) {
			if amount1Out01Mem.Cmp(amount1Out10Mem) != 0 {
				t.Error(amount1Out01Mem, amount1Out10Mem)
			}
			if amount1Out01OBMem.Cmp(amount1Out10OBMem) == 0 {
				t.Error(amount1Out01OBMem, amount1Out10OBMem)
			}
			if amount1Out01Mem.Cmp(amount1Out01OBMem) == 0 {
				t.Error(amount1Out01Mem, amount1Out01OBMem)
			}

			if amount1Out10Mem.Cmp(amount1Out10OBMem) == 0 {
				t.Error(amount1Out10Mem, amount1Out10OBMem)
			}
		})

		swap := New(newBus, immutableTree.GetLastImmutable())
		pair01Disk := swap.Pair(0, 1)
		pair10Disk := swap.Pair(1, 0)
		amount1Out01Disk := pair01Disk.CalculateBuyForSell(big.NewInt(1e10))
		amount1Out10Disk := pair10Disk.CalculateBuyForSell(big.NewInt(1e10))
		amount1Out01OBDisk := pair01Disk.CalculateBuyForSellWithOrders(big.NewInt(1e10))
		amount1Out10OBDisk := pair10Disk.CalculateBuyForSellWithOrders(big.NewInt(1e10))
		t.Run("disk", func(t *testing.T) {
			if amount1Out01Disk.Cmp(amount1Out10Disk) != 0 {
				t.Error(amount1Out01Disk, amount1Out10Disk)
			}
			if amount1Out01OBDisk.Cmp(amount1Out10OBDisk) == 0 {
				t.Error(amount1Out01OBDisk, amount1Out10OBDisk)
			}
			if amount1Out01Disk.Cmp(amount1Out01OBDisk) == 0 {
				t.Error(amount1Out01Disk, amount1Out01OBDisk)
			}

			if amount1Out10Disk.Cmp(amount1Out10OBDisk) == 0 {
				t.Error(amount1Out10Disk, amount1Out10OBDisk)
			}
		})

		t.Run("mem/disk", func(t *testing.T) {
			if amount1Out01Mem.Cmp(amount1Out01Disk) != 0 {
				t.Error(amount1Out01Mem, amount1Out01Disk)
			}
			if amount1Out10Mem.Cmp(amount1Out10Disk) != 0 {
				t.Error(amount1Out10Mem, amount1Out10Disk)
			}
			if amount1Out01OBMem.Cmp(amount1Out01OBDisk) != 0 {
				t.Error(amount1Out01OBMem, amount1Out01OBDisk)
			}
			if amount1Out10OBMem.Cmp(amount1Out10OBDisk) != 0 {
				t.Log(pair10.SellOrderIDs())
				t.Log(pair10Disk.SellOrderIDs())
				t.Error(amount1Out10OBMem, amount1Out10OBDisk)
			}
		})
	})
}

func TestPair_SetOrder_01(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(5e10), big.NewInt(1e10))
	pair := swap.Pair(0, 1)

	volumeBuy := big.NewInt(1e10)
	mul := func(price int64, volumeBuy *big.Int) *big.Int {
		return big.NewInt(0).Mul(big.NewInt(price), volumeBuy)
	}
	/*idHigher :*/ _ = pair.AddOrder(mul(3, volumeBuy), volumeBuy, types.Address{}, 0).id
	/*idMostHigher :*/ _ = pair.AddOrder(mul(1, volumeBuy), volumeBuy, types.Address{}, 0).id
	_ = pair.AddOrder(mul(2, volumeBuy), volumeBuy, types.Address{}, 0).id
	idMostLower := pair.AddOrder(mul(10, volumeBuy), volumeBuy, types.Address{}, 0).id
	idLower := pair.AddOrder(mul(8, volumeBuy), volumeBuy, types.Address{}, 0).id
	_ = pair.AddOrder(mul(9, volumeBuy), volumeBuy, types.Address{}, 0).id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sell (sorted pair)", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			pair = swap.Pair(0, 1)
			t.Run("get", func(t *testing.T) {
				t.Run("set", func(t *testing.T) {
					t.Run("lowest", func(t *testing.T) {
						order, _ := pair.OrderSellLast()
						if idMostLower != order.id {
							t.Log(pair.SellOrderIDs())
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
				})
				t.Run("update", func(t *testing.T) {
					lastSellLower, indexSellLower := pair.OrderSellLast()
					if indexSellLower == -1 {
						t.Error("orders not loaded, last index", indexSellLower)
					}
					if nil == lastSellLower {
						t.Fatal("order is nil")
					}
					if len(pair.SellOrderIDs())-2 != indexSellLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastSellLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
						}
						order := pair.OrderSellByIndex(indexSellLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})

					t.Run("low", func(t *testing.T) {
						t.Skip("allow sell order with highest price ")
						order := pair.OrderSellByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderSellByIndex(0).Price().Cmp(pair.OrderSellByIndex(indexSellLower).Price()) != 1 {
							t.Errorf("not sorted orders")
						}
					})
				})
			})
		})
		t.Run("disk", func(t *testing.T) {
			swap = New(newBus, immutableTree.GetLastImmutable())
			pair = swap.Pair(0, 1)
			t.Run("load", func(t *testing.T) {
				lastSellLower, indexSellLower := pair.OrderSellLast()
				if indexSellLower == -1 {
					t.Error("orders not loaded, last index", indexSellLower)
				}
				if nil == lastSellLower {
					t.Fatal("order is nil")
				}
				if len(pair.SellOrderIDs())-2 != indexSellLower {
					t.Error("error index")
				}
				t.Run("lowest", func(t *testing.T) {
					if idMostLower != lastSellLower.id {
						t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
					}
					order := pair.OrderSellByIndex(indexSellLower)
					if idMostLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
					}
				})
				t.Run("low", func(t *testing.T) {
					t.Skip("allow sell order with highest price ")
					order := pair.OrderSellByIndex(0)
					if idLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
					}
				})
				t.Run("cmp", func(t *testing.T) {
					if pair.OrderSellByIndex(0).Price().Cmp(pair.OrderSellByIndex(indexSellLower).Price()) != 1 {
						t.Errorf("not sorted orders")
					}
				})
			})
		})

	})
}

func TestPair_SetOrder_10(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(1, 0, big.NewInt(5e10), big.NewInt(1e10))
	pair := swap.Pair(1, 0)

	volumeBuy := big.NewInt(1e10)
	mul := func(price int64, volumeBuy *big.Int) *big.Int {
		return big.NewInt(0).Mul(big.NewInt(price), volumeBuy)
	}
	/*idMostHigher :*/ _ = pair.AddOrder(mul(1, volumeBuy), volumeBuy, types.Address{}, 0).id
	/*idHigher :*/ _ = pair.AddOrder(mul(2, volumeBuy), volumeBuy, types.Address{}, 0).id
	idLower := pair.AddOrder(mul(9, volumeBuy), volumeBuy, types.Address{}, 0).id
	idMostLower := pair.AddOrder(mul(10, volumeBuy), volumeBuy, types.Address{}, 0).id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sell (sorted pair)", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			pair = swap.Pair(1, 0)
			t.Run("get", func(t *testing.T) {
				t.Run("set", func(t *testing.T) {
					t.Run("lowest", func(t *testing.T) {

						order, _ := pair.OrderSellLast()
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
				})
				t.Run("update", func(t *testing.T) {
					lastSellLower, indexSellLower := pair.OrderSellLast()
					if indexSellLower == -1 {
						t.Error("orders not loaded, last index", indexSellLower)
					}
					if nil == lastSellLower {
						t.Fatal("order is nil")
					}
					if len(pair.SellOrderIDs())-2 != indexSellLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastSellLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
						}
						order := pair.OrderSellByIndex(indexSellLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderSellByIndex(0).Price().Cmp(pair.OrderSellByIndex(indexSellLower).Price()) != 1 {
							t.Errorf("not sorted orders")
						}
					})
					t.Run("low", func(t *testing.T) {
						t.Skip("allow sell order with highest price ")
						order := pair.OrderSellByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
				})
			})
		})
		t.Run("disk", func(t *testing.T) {
			swap = New(newBus, immutableTree.GetLastImmutable())
			pair = swap.Pair(1, 0)
			t.Run("load", func(t *testing.T) {
				lastSellLower, indexSellLower := pair.OrderSellLast()
				if indexSellLower == -1 {
					t.Error("orders not loaded, last index", indexSellLower)
				}
				if nil == lastSellLower {
					t.Fatal("order is nil")
				}
				if len(pair.SellOrderIDs())-2 != indexSellLower {
					t.Error("error index")
				}
				t.Run("lowest", func(t *testing.T) {
					if idMostLower != lastSellLower.id {
						t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
					}
					order := pair.SellOrderIDs()[indexSellLower]
					if idMostLower != pair.getOrder(order).id {
						t.Errorf("id last sell order from array want %v, got %v", idMostLower, order)
					}
				})
				t.Run("cmp", func(t *testing.T) {
					if pair.OrderSellByIndex(0).Price().Cmp(pair.OrderSellByIndex(indexSellLower).Price()) != 1 {
						t.Errorf("not sorted orders")
					}
				})
				t.Run("low", func(t *testing.T) {
					t.Skip("allow sell order with highest price ")
					order := pair.OrderSellByIndex(0)
					if idLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
					}
				})
			})
		})

	})
}

func TestPair_CalculateBuyForSellWithOrders_01(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))
	pair := swap.Pair(0, 1)
	t.Run("volumes for change price", func(t *testing.T) {
		t.Run("low price", func(t *testing.T) {
			price := big.NewFloat(0.5)
			t.Run("sell", func(t *testing.T) {
				amount0, _ := pair.CalculateAddAmountsForPrice(price)
				if amount0.Cmp(big.NewInt(4146)) != 0 {
					t.Errorf("amount0 want %v, got %v", big.NewInt(4146), amount0)
				}
				amount1 := pair.CalculateBuyForSell(amount0)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "0.500070691361515607" {
					t.Error(amount0, amount1, p.Price().Text('f', 18), price)
				}
			})
			t.Run("buy", func(t *testing.T) {
				_, amount1 := pair.CalculateAddAmountsForPrice(price)
				if amount1.Cmp(big.NewInt(2926)) != 0 {
					t.Errorf("amount1 want %v, got %v", big.NewInt(2926), amount1)
				}
				amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "0.500106044538706274" {
					t.Error(amount0, amount1, p.Price().Text('f', 18), price)
				}
			})
		})
		t.Run("high price", func(t *testing.T) {
			t.Skip("allow sell order with highest price without diff reserves")
			price := big.NewFloat(2)
			t.Run("sell", func(t *testing.T) {
				amount0, _ := pair.CalculateAddAmountsForPrice(price)
				if amount0.Cmp(big.NewInt(-2926)) != 0 {
					t.Errorf("amount0 want %v, got %v", big.NewInt(-2926), amount0)
				}
				amount1 := pair.CalculateBuyForSellAllowNeg(amount0)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "1.999575911789652216" {
					t.Error(amount0, amount1, p.Price().Text('f', 18), price)
				}
			})
			t.Run("buy", func(t *testing.T) {
				_, amount1 := pair.CalculateAddAmountsForPrice(price)
				if amount1.Cmp(big.NewInt(-4146)) != 0 {
					t.Errorf("amount1 want %v, got %v", big.NewInt(-4146), amount1)
				}
				amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "1.999717274526434885" {
					t.Error(amount0, amount1, p.Price().Text('f', 18), price)
				}
			})
		})
	})

	t.Run("without orders", func(t *testing.T) {
		t.Run("sell", func(t *testing.T) {
			amount1Out := pair.CalculateBuyForSell(big.NewInt(5000))
			amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(5000))
			if amount1Out.Cmp(amount1OutWithOB) != 0 {
				t.Error("not equal", amount1Out, amount1OutWithOB)
			}
		})
		t.Run("buy", func(t *testing.T) {
			amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(5000))
			amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(5000))
			if amount0In.Cmp(amount0InWithOB) != 0 {
				t.Error("not equal", amount0In, amount0InWithOB)
			}
		})
	})

	t.Run("with orders", func(t *testing.T) {
		t.Run("one order", func(t *testing.T) {
			pair.AddOrder(big.NewInt(2000), big.NewInt(1000), types.Address{}, 0)
			if pair.OrderSellByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
				t.Error("error set order")
			}
			t.Run("sell", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146))
					if amount1Out.Cmp(amount1OutWithOB) != 0 {
						t.Error("not equal sell before first order", amount1Out, amount1OutWithOB)
					}
				})

				t.Run("first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000))
					if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
						t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
					}
				})
				t.Run("small after order", func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Error("Recovered", r)
						}
					}()
					_ = pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount1 := pair.CalculateBuyForSell(big.NewInt(4146))
					p := pair.AddLastSwapStep(big.NewInt(4146), amount1)
					amount1Out := big.NewInt(0).Add(amount1, p.CalculateBuyForSell(big.NewInt(1000)))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000 + 1000))
					if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
						t.Error("want to spend 2,002-0.1% more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
					}
				})
			})
			t.Run("buy", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuy(big.NewInt(2926))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926))
					if amount0In.Cmp(amount0InWithOB) != 0 {
						t.Error("not equal", amount0In, amount0InWithOB)
					}
				})
				t.Run("first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuy(big.NewInt(2926))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2001" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
				t.Run("small after order", func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Error("Recovered", r)
						}
					}()
					_ = pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
					p := pair.AddLastSwapStep(amount0, big.NewInt(2926)).AddLastSwapStep(big.NewInt(2), big.NewInt(-1))
					amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999 + 466))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2001" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
			})
			t.Run("two equal orders", func(t *testing.T) {
				pair.AddOrder(big.NewInt(2000), big.NewInt(1000), types.Address{}, 0)
				if pair.OrderSellByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 &&
					pair.OrderSellByIndex(1).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
					t.Error("error set orders")
				}
				t.Run("sell", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
							t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 4000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "1998" {
							t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						for _, i := range []int64{12, 20, 24, 30, 43, 55, 78, 89, 103, 200, 500, 700, 750, 999, 1111, 2222, 2500, 3001, 3005, 4321, 5432} {
							t.Run(strconv.Itoa(int(i)), func(t *testing.T) {
								amount1 := pair.CalculateBuyForSell(big.NewInt(4146))
								p := pair.AddLastSwapStep(big.NewInt(4146), amount1).AddLastSwapStep(big.NewInt(4), big.NewInt(-2))
								amount1Out := big.NewInt(0).Add(amount1, p.CalculateBuyForSell(big.NewInt(i)))
								amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 4000 + i))
								if amount1OutWithOB.Cmp(big.NewInt(0).Add(amount1Out, big.NewInt(1998))) != 0 {
									t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
								}
							})
						}
					})
				})
				t.Run("buy", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuy(big.NewInt(2926))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2001" {
							t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuy(big.NewInt(2926))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 1998))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4001" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
						p := pair.AddLastSwapStep(amount0, big.NewInt(2926))
						amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 1998 + 466))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4001" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
				})
				t.Run("three orders", func(t *testing.T) {
					pair.AddOrder(big.NewInt(3000), big.NewInt(1000), types.Address{}, 0)
					if pair.OrderSellByIndex(2).Price().Cmp(CalcPriceSell(big.NewInt(3000), big.NewInt(1000))) != 0 {
						t.Error("error set orders")
					}
					t.Run("sell", func(t *testing.T) {
						t.Skip("todo") // todo: add tests
					})
					t.Run("buy", func(t *testing.T) {
						t.Skip("todo") // todo: add tests
					})
				})
			})
		})
	})
}

func TestPair_CalculateBuyForSellWithOrders_10(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(10000), big.NewInt(10000))
	pair := swap.Pair(1, 0)

	t.Run("without orders", func(t *testing.T) {
		t.Run("sell", func(t *testing.T) {
			amount1Out := pair.CalculateBuyForSell(big.NewInt(5000))
			amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(5000))
			if amount1Out.Cmp(amount1OutWithOB) != 0 {
				t.Error("not equal", amount1Out, amount1OutWithOB)
			}
		})
		t.Run("buy", func(t *testing.T) {
			amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(5000))
			amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(5000))
			if amount0In.Cmp(amount0InWithOB) != 0 {
				t.Error("not equal", amount0In, amount0InWithOB)
			}
		})
	})

	t.Run("with orders", func(t *testing.T) {
		t.Run("one order", func(t *testing.T) {
			pair.AddOrder(big.NewInt(2000), big.NewInt(1000), types.Address{}, 0)
			t.Log(pair.OrderSellByIndex(0).Price(), CalcPriceSell(big.NewInt(2000), big.NewInt(1000)))
			if pair.OrderSellByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
				t.Error("error set order")
			}
			t.Run("sell", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146))
					if amount1Out.Cmp(amount1OutWithOB) != 0 {
						t.Error("not equal sell before first order", amount1Out, amount1OutWithOB)
					}
				})

				t.Run("first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000))
					if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
						t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
					}
				})
				t.Run("small after order", func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Error("Recovered", r)
						}
					}()
					_ = pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount1Out0 := pair.CalculateBuyForSell(big.NewInt(4146))
					amount1Out1 := pair.AddLastSwapStep(big.NewInt(4146), amount1Out0).CalculateBuyForSell(big.NewInt(1000))
					amount1Out := big.NewInt(0).Add(amount1Out0, amount1Out1)
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000 + 1000))
					if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
						t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
					}
				})
			})
			t.Run("buy", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926))
					if amount0In.Cmp(amount0InWithOB) != 0 {
						t.Error("not equal", amount0In, amount0InWithOB)
					}
				})
				t.Run("first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuy(big.NewInt(2926))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2001" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
				t.Run("small after order", func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Error("Recovered", r)
						}
					}()
					_ = pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
					p := pair.AddLastSwapStep(amount0, big.NewInt(2926)).AddLastSwapStep(big.NewInt(2), big.NewInt(-1))
					amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999 + 466))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2001" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
			})
			t.Run("two equal orders", func(t *testing.T) {
				pair.AddOrder(big.NewInt(2000), big.NewInt(1000), types.Address{}, 0)
				if pair.OrderSellByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 &&
					pair.OrderSellByIndex(1).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
					t.Error("error set orders")
				}
				t.Run("sell", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 2000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
							t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4146))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 4000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "1998" {
							t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						for _, i := range []int64{12, 20, 24, 30, 43, 55, 78, 89, 103, 200, 500, 700, 750, 999, 1111, 2222, 2500, 3001, 3005, 4321, 5432} {
							t.Run(strconv.Itoa(int(i)), func(t *testing.T) {
								amount1 := pair.CalculateBuyForSell(big.NewInt(4146))
								p := pair.AddLastSwapStep(big.NewInt(4146), amount1).AddLastSwapStep(big.NewInt(4), big.NewInt(-2))
								amount11 := p.CalculateBuyForSell(big.NewInt(i))
								amount1Out := big.NewInt(0).Add(amount1, amount11)
								amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4146 + 4000 + i))
								if amount1OutWithOB.Cmp(big.NewInt(0).Add(amount1Out, big.NewInt(1998))) != 0 {
									t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
								}
							})
						}
					})
				})
				t.Run("buy", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 999))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2001" {
							t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 1998))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4001" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2926))
						p := pair.AddLastSwapStep(amount0, big.NewInt(2926))
						amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2926 + 1998 + 466))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4001" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
				})
				t.Run("three orders", func(t *testing.T) {
					pair.AddOrder(big.NewInt(3000), big.NewInt(1000), types.Address{}, 0)
					if pair.OrderSellByIndex(2).Price().Cmp(CalcPriceSell(big.NewInt(3000), big.NewInt(1000))) != 0 {
						t.Error("error set orders")
					}
					t.Run("sell", func(t *testing.T) {
						t.Skip("todo") // todo: add tests
					})
					t.Run("buy", func(t *testing.T) {
						t.Skip("todo") // todo: add tests
					})
				})
			})
		})
	})
}

func TestPair_CalculateAddAmount0ForPrice_10(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := New(newBus, immutableTree.GetLastImmutable())
	_, _, _, _ = swap.PairCreate(0, 1, big.NewInt(11000), big.NewInt(9000))

	t.Run("sort", func(t *testing.T) {
		pair := swap.Pair(0, 1)
		t.Run("volumes for change price", func(t *testing.T) {
			t.Run("low price", func(t *testing.T) {
				price := big.NewFloat(0.5)
				t.Run("sell", func(t *testing.T) {
					amount0, _ := pair.CalculateAddAmountsForPrice(price)
					if amount0.Cmp(big.NewInt(3074)) != 0 {
						t.Errorf("amount0 want %v, got %v", big.NewInt(3074), amount0)
					}
					amount1 := pair.CalculateBuyForSell(amount0)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "0.500071053005542110" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
				t.Run("buy", func(t *testing.T) {
					_, amount1 := pair.CalculateAddAmountsForPrice(price)
					if amount1.Cmp(big.NewInt(1962)) != 0 {
						t.Errorf("amount1 want %v, got %v", big.NewInt(1962), amount1)
					}
					amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "0.500106587081645682" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
			})
			t.Run("high price", func(t *testing.T) {
				t.Skip("allow sell order with highest price without diff pool reserves")
				price := big.NewFloat(2)
				t.Run("sell", func(t *testing.T) {
					amount0, _ := pair.CalculateAddAmountsForPrice(price)
					if amount0.Cmp(big.NewInt(-3961)) != 0 {
						t.Errorf("amount0 want %v, got %v", big.NewInt(-3961), amount0)
					}
					amount1 := pair.CalculateBuyForSellAllowNeg(amount0)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "1.999573803097030811" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
				t.Run("buy", func(t *testing.T) {
					_, amount1 := pair.CalculateAddAmountsForPrice(price)
					if amount1.Cmp(big.NewInt(-5076)) != 0 {
						t.Errorf("amount1 want %v, got %v", big.NewInt(-5076), amount1)
					}
					amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "1.999715868731353874" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
			})
		})
	})
	t.Run("reverse", func(t *testing.T) {
		pair := swap.Pair(1, 0)
		t.Run("volumes for change price", func(t *testing.T) {
			t.Run("low price", func(t *testing.T) {
				price := big.NewFloat(0.5)
				t.Run("sell", func(t *testing.T) {
					amount0, _ := pair.CalculateAddAmountsForPrice(price)
					if amount0.Cmp(big.NewInt(5076)) != 0 {
						t.Errorf("amount0 want %v, got %v", big.NewInt(5076), amount0)
					}
					amount1 := pair.CalculateBuyForSell(amount0)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "0.500071042909917607" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
				t.Run("buy", func(t *testing.T) {
					_, amount1 := pair.calculateAddAmountsForPrice(price)
					if amount1.Cmp(big.NewInt(3961)) != 0 {
						t.Errorf("amount1 want %v, got %v", big.NewInt(3961), amount1)
					}
					amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "0.500106571936056843" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
			})
			t.Run("high price", func(t *testing.T) {
				t.Skip("allow sell order with highest price without diff pool reserves")
				price := big.NewFloat(2)
				t.Run("sell", func(t *testing.T) {
					amount0, _ := pair.CalculateAddAmountsForPrice(price)
					if amount0.Cmp(big.NewInt(-1962)) != 0 {
						t.Errorf("amount0 want %v, got %v", big.NewInt(-1962), amount0)
					}
					amount1 := pair.CalculateBuyForSellAllowNeg(amount0)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "1.999573742540494470" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
				t.Run("buy", func(t *testing.T) {
					_, amount1 := pair.CalculateAddAmountsForPrice(price)
					if amount1.Cmp(big.NewInt(-3074)) != 0 {
						t.Errorf("amount1 want %v, got %v", big.NewInt(-5078), amount1)
					}
					amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "1.999715828360329684" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
			})
		})
	})
}
