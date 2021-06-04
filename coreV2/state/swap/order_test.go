package swap

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/tendermint/go-amino"
	db "github.com/tendermint/tm-db"
)

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
	order := pair.SetOrder(wantBuy, wantSell, types.Address{})

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	amount0In := helpers.StringToBigInt("10000000000000000000000000")
	amount1Out, orders := pair.calculateBuyForSellWithOrders(amount0In)
	commission0orders, commission1orders, amount0, amount1, _ := CalcDiffPool(amount0In, amount1Out, orders)

	if amount1Out.String() != "14989949147713228731141" {
		t.Error("err")
	}

	amount0ForPrice := pair.CalculateAddAmount0ForPrice(order.Price())
	amount1ForPrice := pair.CalculateBuyForSell(amount0ForPrice)

	if amount0ForPrice.String() != "7330916069244653273088" {
		t.Error("err")
	}
	if amount1ForPrice.String() != "4225079013582808516163" {
		t.Error("err")
	}

	if commission0orders.String() != "14985014985014985015" {
		t.Error("err")
	}
	if commission1orders.String() != "4995004995004995005" {
		t.Error("err")
	}

	amount0AfterPrice := big.NewInt(0).Sub(amount0, amount0ForPrice)
	amount1AfterPrice := big.NewInt(0).Sub(amount1, amount1ForPrice)

	if amount0AfterPrice.String() != "9977669083930755346726912" {
		t.Error("err")
	}
	if amount1AfterPrice.String() != "5769865139125425209983" {
		t.Error("err")
	}

	pair.Swap(amount0ForPrice, big.NewInt(0), big.NewInt(0), amount1ForPrice)
	pair.update(amount0ForPrice, amount1ForPrice)
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
		id := pair.SetOrder(big.NewInt(1), big.NewInt(1), types.Address{}).id
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
		if id := pair.SetOrder(big.NewInt(2), big.NewInt(1), types.Address{}).id; id != 2 {
			t.Errorf("next orders ID want %d, got %d", 2, id)
		}
		if id := pair.SetOrder(big.NewInt(3), big.NewInt(1), types.Address{}).id; id != 3 {
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
		if id := pair.SetOrder(big.NewInt(4), big.NewInt(1), types.Address{}).id; id != 4 {
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
	pair.SetOrder(big.NewInt(15000), big.NewInt(5000), types.Address{})
	pair.SetOrder(big.NewInt(20), big.NewInt(5), types.Address{})

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(0, 1)
	price := pair.OrderSellLowerByIndex(0).Price()

	addAmount0ForPrice := pair.CalculateAddAmount0ForPrice(price)
	if addAmount0ForPrice.Cmp(big.NewInt(7330)) != 0 {
		t.Error("a", addAmount0ForPrice)
	}

	addAmount1 := pair.CalculateBuyForSell(addAmount0ForPrice)
	if addAmount1.Cmp(big.NewInt(4224)) != 0 {
		t.Error("z", addAmount1)
	}

	t.Run("add amount1", func(t *testing.T) {
		addAmount1ForPrice := pair.CalculateAddAmount1ForPrice(price)
		if addAmount1ForPrice.Cmp(addAmount1) != 0 {
			t.Error("a", addAmount1ForPrice)
		}

		addAmount0 := pair.CalculateSellForBuy(addAmount1ForPrice)
		if addAmount0.Cmp(addAmount0ForPrice) != 0 {
			t.Skipf("todo, addAmount0 %s, addAmount0ForPrice %s", addAmount0, addAmount0ForPrice)
		}
	})

	amount0In, owners, _ := pair.BuyWithOrders(big.NewInt(0).Add(addAmount1, big.NewInt(3000)))
	t.Run("owner", func(t *testing.T) {
		if len(owners) != 1 {
			t.Fatal("b", owners)
		}

		if owners[types.Address{}].Cmp(big.NewInt(9000)) != 0 {
			t.Error("c", owners[types.Address{}])
		}
	})

	if amount0In.Cmp(big.NewInt(7330+9009)) != 0 {
		t.Error("d", amount0In)
	}

	t.Run("resort dirty", func(t *testing.T) {
		order := pair.OrderSellLowerByIndex(0)
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

			if owners[types.Address{}].Cmp(big.NewInt(3000)) != 0 {
				t.Error("c", owners[types.Address{}])
			}
		})

		if amount0In.Cmp(big.NewInt(3003)) != 0 {
			t.Error("d", amount0In)
		}

		if pair.OrderSellLowerByIndex(0).id != 1 {
			t.Errorf("want %d, got %d", 1, pair.OrderSellLowerByIndex(0).id)
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
	pair.SetOrder(big.NewInt(15000), big.NewInt(5000), types.Address{})
	pair.SetOrder(big.NewInt(20), big.NewInt(5), types.Address{})

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(0, 1)
	price := pair.OrderSellLowerByIndex(0).Price()
	addAmount0ForPrice := pair.CalculateAddAmount0ForPrice(price)
	if addAmount0ForPrice.Cmp(big.NewInt(7330)) != 0 {
		t.Error("a", addAmount0ForPrice)
	}

	addAmount1 := pair.CalculateBuyForSell(addAmount0ForPrice)
	if addAmount1.Cmp(big.NewInt(4224)) != 0 {
		t.Error("z", addAmount0ForPrice)
	}

	amount1Out, owners, _ := pair.SellWithOrders(big.NewInt(0).Add(addAmount0ForPrice, big.NewInt(9009)))
	if len(owners) != 1 {
		t.Error("b", owners)
	}

	if owners[types.Address{}].Cmp(big.NewInt(9000)) != 0 {
		t.Error("c", owners[types.Address{}])
	}

	if amount1Out.Cmp(big.NewInt(4224+3000)) != 0 {
		t.Error("d", amount1Out)
	}

	t.Run("resort dirty", func(t *testing.T) {
		order := pair.OrderSellLowerByIndex(0)
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

		if pair.OrderSellLowerByIndex(0).id != 1 {
			t.Errorf("want %d, got %d", 1, pair.OrderSellLowerByIndex(0).id)
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
	pair.SetOrder(big.NewInt(15000), big.NewInt(5000), types.Address{})
	pair.SetOrder(big.NewInt(20), big.NewInt(5), types.Address{})

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	pair = swap.Pair(1, 0)
	price := pair.OrderSellLowerByIndex(0).Price()
	addAmount0ForPrice := pair.CalculateAddAmount0ForPrice(price)
	if addAmount0ForPrice.Cmp(big.NewInt(7330)) != 0 {
		t.Error("a", addAmount0ForPrice)
	}

	addAmount1 := pair.CalculateBuyForSell(addAmount0ForPrice)
	if addAmount1.Cmp(big.NewInt(4224)) != 0 {
		t.Error("z", addAmount1)
	}

	amount1Out, owners, _ := pair.SellWithOrders(big.NewInt(0).Add(addAmount0ForPrice, big.NewInt(9009)))
	if len(owners) != 1 {
		t.Error("b", owners)
	}

	if owners[types.Address{}].Cmp(big.NewInt(9000)) != 0 {
		t.Error("c", owners[types.Address{}])
	}

	if amount1Out.Cmp(big.NewInt(4224+3000)) != 0 {
		t.Error("d", amount1Out)
	}

	t.Run("resort dirty", func(t *testing.T) {
		order := pair.OrderSellLowerByIndex(0)
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

		orderNext := pair.OrderSellLowerByIndex(0)
		if order.Price().Cmp(orderNext.Price()) != 1 {
			t.Errorf("order %d price %v, and %d price %v", orderNext.id, orderNext.Price(), order.id, order.Price())
		}
		if orderNext.id != 2 {
			t.Errorf("want %d, got %d", 2, order.id)
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
		if len(appState.Pools[0].Orders) != 2 {
			t.Fatalf("orders are empty, %s", jsonBytes)
		} else {
			t.Logf("%#v", appState.Pools[0].Orders[0])
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
	swap.PairAddOrder(0, 1, big.NewInt(2000), big.NewInt(1000), owner)
	if pair.OrderSellLowerByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
		t.Error("error set order")
	}
	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("order", func(t *testing.T) {
		swap := New(newBus, immutableTree.GetLastImmutable())
		pair := swap.Pair(0, 1)
		amount1Out := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000))
		_, orders := pair.calculateBuyForSellWithOrders(big.NewInt(4147 + 2000))
		if len(orders) == 0 {
			t.Error("empty orders")
		}
		amount1OutSell, owners, _ := pair.SellWithOrders(big.NewInt(4147 + 2000))
		if amount1OutSell.Cmp(amount1Out) != 0 {
			t.Error("not equal", amount1Out, amount1OutSell)
		}
		if len(owners) == 0 {
			t.Error("empty owners")
		}
		if owners[owner] == nil || owners[owner].Cmp(big.NewInt(1998)) != 0 {
			t.Errorf("%#v", owners[owner])
		}
		t.Run("unset", func(t *testing.T) {
			if len(pair.SellLowerOrders()) != 0 {
				t.Errorf("slice len %d, want empty", len(pair.SellLowerOrders()))
			}
		})

		// t.Logf("price %v", pair.Price()) // todo: check error of skipping orders

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
	swap.PairAddOrder(0, 1, big.NewInt(2000), big.NewInt(1000), owner)
	if pair.OrderSellLowerByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
		t.Error("error set order")
	}

	t.Run("order", func(t *testing.T) {
		amount1Out := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 1000))
		_, orders := pair.calculateBuyForSellWithOrders(big.NewInt(4147 + 1000))
		if len(orders) == 0 {
			t.Error("empty orders")
		}
		amount1OutSell, owners, _ := pair.SellWithOrders(big.NewInt(4147 + 1000))
		if amount1OutSell.Cmp(amount1Out) != 0 {
			t.Error("not equal", amount1Out, amount1OutSell)
		}
		if len(owners) == 0 {
			t.Error("empty owners")
		}
		if owners[owner] == nil || owners[owner].Cmp(big.NewInt(999)) != 0 {
			t.Errorf("%#v", owners[owner])
		}
		t.Logf("price %v", pair.Price()) // todo: check error of skipping orders

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
	pair01.SetOrder(big.NewInt(10010), big.NewInt(10000), types.Address{})
	pair01.SetOrder(big.NewInt(10020), big.NewInt(10000), types.Address{})

	pair10 := swap.Pair(1, 0)
	pair10.SetOrder(big.NewInt(1003), big.NewInt(1000), types.Address{})
	pair10.SetOrder(big.NewInt(1004), big.NewInt(1000), types.Address{})

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
	idHigher := pair.SetOrder(mul(3, volumeBuy), volumeBuy, types.Address{}).id
	idMostHigher := pair.SetOrder(mul(1, volumeBuy), volumeBuy, types.Address{}).id
	_ = pair.SetOrder(mul(2, volumeBuy), volumeBuy, types.Address{}).id
	idMostLower := pair.SetOrder(mul(10, volumeBuy), volumeBuy, types.Address{}).id
	idLower := pair.SetOrder(mul(8, volumeBuy), volumeBuy, types.Address{}).id
	_ = pair.SetOrder(mul(9, volumeBuy), volumeBuy, types.Address{}).id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sell (sorted pair)", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			pair = swap.Pair(0, 1)
			t.Run("get", func(t *testing.T) {
				t.Run("set", func(t *testing.T) {
					t.Run("low", func(t *testing.T) {
						order := pair.OrderSellLowerByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.OrderSellLowerByIndex(len(pair.SellLowerOrders()) - 1)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("high", func(t *testing.T) {
						order := pair.sellHigherOrders()[0]
						if idHigher != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idHigher, order.id)
						}
					})
					t.Run("highest", func(t *testing.T) {
						order := pair.sellHigherOrders()[len(pair.sellHigherOrders())-1]
						if idMostHigher != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostHigher, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(len(pair.SellLowerOrders())-1).Price()) != 1 {
							t.Errorf("not sorted orders")
						}
					})
				})
				t.Run("update", func(t *testing.T) {
					lastSellLower, indexSellLower := pair.OrderSellLowerLast()
					if indexSellLower == -1 {
						t.Error("orders not loaded, last index", indexSellLower)
					}
					if nil == lastSellLower {
						t.Fatal("order is nil")
					}
					if len(pair.SellLowerOrders())-1 != indexSellLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastSellLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
						}
						order := pair.OrderSellLowerByIndex(indexSellLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.OrderSellLowerByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(indexSellLower).Price()) != 1 {
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
				lastSellLower, indexSellLower := pair.OrderSellLowerLast()
				if indexSellLower == -1 {
					t.Error("orders not loaded, last index", indexSellLower)
				}
				if nil == lastSellLower {
					t.Fatal("order is nil")
				}
				if len(pair.SellLowerOrders())-1 != indexSellLower {
					t.Error("error index")
				}
				t.Run("lowest", func(t *testing.T) {
					if idMostLower != lastSellLower.id {
						t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
					}
					order := pair.OrderSellLowerByIndex(indexSellLower)
					if idMostLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
					}
				})
				t.Run("low", func(t *testing.T) {
					order := pair.OrderSellLowerByIndex(0)
					if idLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
					}
				})
				t.Run("cmp", func(t *testing.T) {
					if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(indexSellLower).Price()) != 1 {
						t.Errorf("not sorted orders")
					}
				})
			})
		})

	})
	t.Run("buy (unsorted pair)", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			pair = swap.Pair(1, 0)
			t.Run("get", func(t *testing.T) {
				t.Run("set", func(t *testing.T) {
					t.Run("low", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(len(pair.BuyHigherOrders()) - 1)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderBuyHigherByIndex(0).Price().Cmp(pair.OrderBuyHigherByIndex(len(pair.BuyHigherOrders())-1).Price()) != -1 {
							t.Errorf("not sorted orders")
						}
					})
				})
				t.Run("update", func(t *testing.T) {
					lastBuyLower, indexBuyLower := pair.OrderBuyHigherLast()
					if indexBuyLower == -1 {
						t.Error("orders not loaded, last index", indexBuyLower)
					}
					if nil == lastBuyLower {
						t.Fatal("order is nil")
					}
					if len(pair.BuyHigherOrders())-1 != indexBuyLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastBuyLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastBuyLower.id)
						}
						order := pair.OrderBuyHigherByIndex(indexBuyLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderBuyHigherByIndex(0).Price().Cmp(pair.OrderBuyHigherByIndex(indexBuyLower).Price()) != -1 {
							t.Errorf("not sorted orders")
						}
					})
				})
			})
		})
		t.Run("disk", func(t *testing.T) {
			swap = New(newBus, immutableTree.GetLastImmutable())
			pair = swap.Pair(1, 0)
			t.Run("get", func(t *testing.T) {
				t.Run("load", func(t *testing.T) {
					lastBuyLower, indexBuyLower := pair.OrderBuyHigherLast()
					if indexBuyLower == -1 {
						t.Error("orders not loaded, last index", indexBuyLower)
					}
					if nil == lastBuyLower {
						t.Fatal("order is nil")
					}
					if len(pair.BuyHigherOrders())-1 != indexBuyLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastBuyLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastBuyLower.id)
						}
						order := pair.OrderBuyHigherByIndex(indexBuyLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderBuyHigherByIndex(0).Price().Cmp(pair.OrderBuyHigherByIndex(indexBuyLower).Price()) != -1 {
							t.Errorf("not sorted orders")
						}
					})
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
	idMostHigher := pair.SetOrder(mul(1, volumeBuy), volumeBuy, types.Address{}).id
	idHigher := pair.SetOrder(mul(2, volumeBuy), volumeBuy, types.Address{}).id
	idLower := pair.SetOrder(mul(9, volumeBuy), volumeBuy, types.Address{}).id
	idMostLower := pair.SetOrder(mul(10, volumeBuy), volumeBuy, types.Address{}).id

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("sell (sorted pair)", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			pair = swap.Pair(1, 0)
			t.Run("get", func(t *testing.T) {
				t.Run("set", func(t *testing.T) {
					t.Run("low", func(t *testing.T) {
						order := pair.OrderSellLowerByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.OrderSellLowerByIndex(len(pair.SellLowerOrders()) - 1)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("high", func(t *testing.T) {
						order := pair.sellHigherOrders()[0]
						if idHigher != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idHigher, order.id)
						}
					})
					t.Run("highest", func(t *testing.T) {
						order := pair.sellHigherOrders()[len(pair.sellHigherOrders())-1]
						if idMostHigher != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostHigher, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(len(pair.SellLowerOrders())-1).Price()) != 1 {
							t.Errorf("not sorted orders")
						}
					})
				})
				t.Run("update", func(t *testing.T) {
					lastSellLower, indexSellLower := pair.OrderSellLowerLast()
					if indexSellLower == -1 {
						t.Error("orders not loaded, last index", indexSellLower)
					}
					if nil == lastSellLower {
						t.Fatal("order is nil")
					}
					if len(pair.SellLowerOrders())-1 != indexSellLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastSellLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
						}
						order := pair.OrderSellLowerByIndex(indexSellLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.OrderSellLowerByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(indexSellLower).Price()) != 1 {
							t.Errorf("not sorted orders")
						}
					})
				})
			})
		})
		t.Run("disk", func(t *testing.T) {
			swap = New(newBus, immutableTree.GetLastImmutable())
			pair = swap.Pair(1, 0)
			t.Run("load", func(t *testing.T) {
				lastSellLower, indexSellLower := pair.OrderSellLowerLast()
				if indexSellLower == -1 {
					t.Error("orders not loaded, last index", indexSellLower)
				}
				if nil == lastSellLower {
					t.Fatal("order is nil")
				}
				if len(pair.SellLowerOrders())-1 != indexSellLower {
					t.Error("error index")
				}
				t.Run("lowest", func(t *testing.T) {
					if idMostLower != lastSellLower.id {
						t.Errorf("id not equal, want %v, got %v", idMostLower, lastSellLower.id)
					}
					order := pair.SellLowerOrders()[indexSellLower]
					if idMostLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
					}
				})
				t.Run("low", func(t *testing.T) {
					order := pair.OrderSellLowerByIndex(0)
					if idLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
					}
				})
				t.Run("cmp", func(t *testing.T) {
					if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(indexSellLower).Price()) != 1 {
						t.Errorf("not sorted orders")
					}
				})
			})
		})

	})
	t.Run("buy (unsorted pair)", func(t *testing.T) {
		t.Run("mem", func(t *testing.T) {
			pair = swap.Pair(0, 1)
			t.Run("get", func(t *testing.T) {
				t.Run("set", func(t *testing.T) {
					t.Run("low", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(len(pair.BuyHigherOrders()) - 1)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderBuyHigherByIndex(0).Price().Cmp(pair.OrderBuyHigherByIndex(len(pair.BuyHigherOrders())-1).Price()) != -1 {
							t.Errorf("not sorted orders")
						}
					})
				})
				t.Run("update", func(t *testing.T) {
					lastBuyLower, indexBuyLower := pair.OrderBuyHigherLast()
					if indexBuyLower == -1 {
						t.Error("orders not loaded, last index", indexBuyLower)
					}
					if nil == lastBuyLower {
						t.Fatal("order is nil")
					}
					if len(pair.BuyHigherOrders())-1 != indexBuyLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastBuyLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastBuyLower.id)
						}
						order := pair.OrderBuyHigherByIndex(indexBuyLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("cmp", func(t *testing.T) {
						if pair.OrderBuyHigherByIndex(0).Price().Cmp(pair.OrderBuyHigherByIndex(indexBuyLower).Price()) != -1 {
							t.Errorf("not sorted orders")
						}
					})
				})
			})
		})
		t.Run("disk", func(t *testing.T) {
			swap = New(newBus, immutableTree.GetLastImmutable())
			pair = swap.Pair(0, 1)
			t.Run("get", func(t *testing.T) {
				t.Run("load", func(t *testing.T) {
					lastBuyLower, indexBuyLower := pair.OrderBuyHigherLast()
					if indexBuyLower == -1 {
						t.Error("orders not loaded, last index", indexBuyLower)
					}
					if nil == lastBuyLower {
						t.Fatal("order is nil")
					}
					if len(pair.BuyHigherOrders())-1 != indexBuyLower {
						t.Error("error index")
					}
					t.Run("lowest", func(t *testing.T) {
						if idMostLower != lastBuyLower.id {
							t.Errorf("id not equal, want %v, got %v", idMostLower, lastBuyLower.id)
						}
						order := pair.OrderBuyHigherByIndex(indexBuyLower)
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.OrderBuyHigherByIndex(0)
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
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
				amount0 := pair.CalculateAddAmount0ForPrice(price)
				if amount0.Cmp(big.NewInt(4147)) != 0 {
					t.Errorf("amount0 want %v, got %v", big.NewInt(4147), amount0)
				}
				amount1 := pair.CalculateBuyForSell(amount0)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "0.499964656817699860" {
					t.Error(amount0, amount1, p.Price(), price)
				}
			})
			t.Run("buy", func(t *testing.T) {
				amount1 := pair.CalculateAddAmount1ForPrice(price)
				if amount1.Cmp(big.NewInt(2927)) != 0 {
					t.Errorf("amount1 want %v, got %v", big.NewInt(2927), amount1)
				}
				amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "0.499964656817699860" {
					t.Error(amount0, amount1, p.Price(), price)
				}
			})
		})
		t.Run("high price", func(t *testing.T) {
			price := big.NewFloat(2)
			t.Run("sell", func(t *testing.T) {
				amount0 := pair.CalculateAddAmount0ForPrice(price)
				if amount0.Cmp(big.NewInt(-2927)) != 0 {
					t.Errorf("amount0 want %v, got %v", big.NewInt(-2927), amount0)
				}
				amount1 := pair.CalculateBuyForSellAllowNeg(amount0)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "2.000141382723031214" {
					t.Error(amount0, amount1, p.Price().Text('f', 18), price)
				}
			})
			t.Run("buy", func(t *testing.T) {
				amount1 := pair.CalculateAddAmount1ForPrice(price)
				if amount1.Cmp(big.NewInt(-4147)) != 0 {
					t.Errorf("amount1 want %v, got %v", big.NewInt(-4147), amount1)
				}
				amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
				p := pair.AddLastSwapStep(amount0, amount1)
				if p.Price().Text('f', 18) != "2.000141382723031214" {
					t.Error(amount0, amount1, p.Price(), price)
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
			pair.SetOrder(big.NewInt(2000), big.NewInt(1000), types.Address{})
			if pair.OrderSellLowerByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
				t.Error("error set order")
			}
			t.Run("sell", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147))
					if amount1Out.Cmp(amount1OutWithOB) != 0 {
						t.Error("not equal sell before first order", amount1Out, amount1OutWithOB)
					}
				})

				t.Run("first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000))
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
					_ = pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount1 := pair.CalculateBuyForSell(big.NewInt(4147))
					p := pair.AddLastSwapStep(big.NewInt(4147), amount1)
					amount1Out := big.NewInt(0).Add(amount1, p.CalculateBuyForSell(big.NewInt(1000)))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000 + 1000))
					if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
						t.Error("want to spend 2,002-0.1% more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
					}
				})
			})
			t.Run("buy", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927))
					if amount0In.Cmp(amount0InWithOB) != 0 {
						t.Error("not equal", amount0In, amount0InWithOB)
					}
				})
				t.Run("first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
				t.Run("small after order", func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Error("Recovered", r)
						}
					}()
					_ = pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
					p := pair.AddLastSwapStep(amount0, big.NewInt(2927))
					amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999 + 466))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
			})
			t.Run("two equal orders", func(t *testing.T) {
				pair.SetOrder(big.NewInt(2000), big.NewInt(1000), types.Address{})
				if pair.OrderSellLowerByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 &&
					pair.OrderSellLowerByIndex(1).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
					t.Error("error set orders")
				}
				t.Run("sell", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
							t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 4000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "1998" {
							t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						for _, i := range []int64{12, 20, 24, 30, 43, 55, 78, 89, 103, 200, 500, 700, 750, 999, 1111, 2222, 2500, 3001, 3005, 4321, 5432} {
							t.Run(strconv.Itoa(int(i)), func(t *testing.T) {
								amount1 := pair.CalculateBuyForSell(big.NewInt(4147))
								p := pair.AddLastSwapStep(big.NewInt(4147), amount1)
								amount1Out := big.NewInt(0).Add(amount1, p.CalculateBuyForSell(big.NewInt(i)))
								amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 4000 + i))
								if amount1OutWithOB.Cmp(big.NewInt(0).Add(amount1Out, big.NewInt(1998))) != 0 {
									t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
								}
							})
						}
					})
				})
				t.Run("buy", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuy(big.NewInt(2927))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
							t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuy(big.NewInt(2927))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 1998))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4000" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
						p := pair.AddLastSwapStep(amount0, big.NewInt(2927))
						amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 1998 + 466))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4000" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
				})
				t.Run("three orders", func(t *testing.T) {
					pair.SetOrder(big.NewInt(3000), big.NewInt(1000), types.Address{})
					if pair.OrderSellLowerByIndex(2).Price().Cmp(CalcPriceSell(big.NewInt(3000), big.NewInt(1000))) != 0 {
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
			pair.SetOrder(big.NewInt(2000), big.NewInt(1000), types.Address{})
			t.Log(pair.OrderSellLowerByIndex(0).Price(), CalcPriceSell(big.NewInt(2000), big.NewInt(1000)))
			if pair.OrderSellLowerByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
				t.Error("error set order")
			}
			t.Run("sell", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147))
					if amount1Out.Cmp(amount1OutWithOB) != 0 {
						t.Error("not equal sell before first order", amount1Out, amount1OutWithOB)
					}
				})

				t.Run("first order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000))
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
					_ = pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount1Out := pair.CalculateBuyForSell(big.NewInt(4147 + 1000))
					amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000 + 1000))
					if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
						t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
					}
				})
			})
			t.Run("buy", func(t *testing.T) {
				t.Run("before first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927))
					if amount0In.Cmp(amount0InWithOB) != 0 {
						t.Error("not equal", amount0In, amount0InWithOB)
					}
				})
				t.Run("first order", func(t *testing.T) {
					amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
				t.Run("small after order", func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Error("Recovered", r)
						}
					}()
					_ = pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999 + 1))
				})
				t.Run("more order", func(t *testing.T) {
					amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
					p := pair.AddLastSwapStep(amount0, big.NewInt(2927))
					amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
					amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999 + 466))
					if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
						t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
					}
				})
			})
			t.Run("two equal orders", func(t *testing.T) {
				pair.SetOrder(big.NewInt(2000), big.NewInt(1000), types.Address{})
				if pair.OrderSellLowerByIndex(0).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 &&
					pair.OrderSellLowerByIndex(1).Price().Cmp(CalcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
					t.Error("error set orders")
				}
				t.Run("sell", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 2000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "999" {
							t.Error("want to spend 2,000 more and get 1,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount1Out := pair.CalculateBuyForSell(big.NewInt(4147))
						amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 4000))
						if big.NewInt(0).Sub(amount1OutWithOB, amount1Out).String() != "1998" {
							t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						for _, i := range []int64{12, 20, 24, 30, 43, 55, 78, 89, 103, 200, 500, 700, 750, 999, 1111, 2222, 2500, 3001, 3005, 4321, 5432} {
							t.Run(strconv.Itoa(int(i)), func(t *testing.T) {
								amount1 := pair.CalculateBuyForSell(big.NewInt(4147))
								p := pair.AddLastSwapStep(big.NewInt(4147), amount1)
								amount1Out := big.NewInt(0).Add(amount1, p.CalculateBuyForSell(big.NewInt(i)))
								amount1OutWithOB := pair.CalculateBuyForSellWithOrders(big.NewInt(4147 + 4000 + i))
								if amount1OutWithOB.Cmp(big.NewInt(0).Add(amount1Out, big.NewInt(1998))) != 0 {
									t.Error("want to spend 4,000 more and get 2,000-0.1% more by order", amount1Out, amount1OutWithOB)
								}
							})
						}
					})
				})
				t.Run("buy", func(t *testing.T) {
					t.Run("before second order", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
							t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 1998))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4000" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
					t.Run("all orders and more", func(t *testing.T) {
						amount0 := pair.CalculateSellForBuyAllowNeg(big.NewInt(2927))
						p := pair.AddLastSwapStep(amount0, big.NewInt(2927))
						amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 1998 + 466))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4000" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
				})
				t.Run("three orders", func(t *testing.T) {
					pair.SetOrder(big.NewInt(3000), big.NewInt(1000), types.Address{})
					if pair.OrderSellLowerByIndex(2).Price().Cmp(CalcPriceSell(big.NewInt(3000), big.NewInt(1000))) != 0 {
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
					amount0 := pair.CalculateAddAmount0ForPrice(price)
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
					amount1 := pair.CalculateAddAmount1ForPrice(price)
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
				price := big.NewFloat(2)
				t.Run("sell", func(t *testing.T) {
					amount0 := pair.CalculateAddAmount0ForPrice(price)
					if amount0.Cmp(big.NewInt(-3962)) != 0 {
						t.Errorf("amount0 want %v, got %v", big.NewInt(-3962), amount0)
					}
					amount1 := pair.CalculateBuyForSellAllowNeg(amount0)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "2.000142085819835103" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
				t.Run("buy", func(t *testing.T) {
					amount1 := pair.CalculateAddAmount1ForPrice(price)
					if amount1.Cmp(big.NewInt(-5078)) != 0 {
						t.Errorf("amount1 want %v, got %v", big.NewInt(-5078), amount1)
					}
					amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "2.000284171639670427" {
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
					amount0 := pair.CalculateAddAmount0ForPrice(price)
					if amount0.Cmp(big.NewInt(5078)) != 0 {
						t.Errorf("amount0 want %v, got %v", big.NewInt(3074), amount0)
					}
					amount1 := pair.CalculateBuyForSell(amount0)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "0.499928967182838468" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
				t.Run("buy", func(t *testing.T) {
					amount1 := pair.CalculateAddAmount1ForPrice(price)
					if amount1.Cmp(big.NewInt(3962)) != 0 {
						t.Errorf("amount1 want %v, got %v", big.NewInt(3962), amount1)
					}
					amount0 := pair.CalculateSellForBuyAllowNeg(amount1)
					p := pair.AddLastSwapStep(amount0, amount1)
					if p.Price().Text('f', 18) != "0.499964481068409461" {
						t.Error(amount0, amount1, p.Price().Text('f', 18), price)
					}
				})
			})
			t.Run("high price", func(t *testing.T) {
				price := big.NewFloat(2)
				t.Run("sell", func(t *testing.T) {
					amount0 := pair.CalculateAddAmount0ForPrice(price)
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
					amount1 := pair.CalculateAddAmount1ForPrice(price)
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
