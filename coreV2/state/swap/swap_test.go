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
	idMostHigher := pair.SetOrder(mul(1, volumeBuy), volumeBuy)
	idHigher := pair.SetOrder(mul(2, volumeBuy), volumeBuy)
	idLower := pair.SetOrder(mul(9, volumeBuy), volumeBuy)
	idMostLower := pair.SetOrder(mul(10, volumeBuy), volumeBuy)

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
						order := pair.SellLowerOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.SellLowerOrders()[len(pair.SellLowerOrders())-1]
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
						order := pair.SellLowerOrders()[indexSellLower]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.SellLowerOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
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
					order := pair.SellLowerOrders()[indexSellLower]
					if idMostLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
					}
				})
				t.Run("low", func(t *testing.T) {
					order := pair.SellLowerOrders()[0]
					if idLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
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
						order := pair.BuyHigherOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.BuyHigherOrders()[len(pair.BuyHigherOrders())-1]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
				})
				t.Run("cmp", func(t *testing.T) {
					t.Skip("== 1 or == -1")
					if pair.BuyHigherOrders()[0].Price().Cmp(pair.BuyHigherOrders()[len(pair.BuyHigherOrders())-1].Price()) != 1 {
						t.Errorf("1")
					}
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
						order := pair.BuyHigherOrders()[indexBuyLower]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.BuyHigherOrders()[0]
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
						order := pair.BuyHigherOrders()[indexBuyLower]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.BuyHigherOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
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
	idMostHigher := pair.SetOrder(mul(1, volumeBuy), volumeBuy)
	idHigher := pair.SetOrder(mul(2, volumeBuy), volumeBuy)
	idLower := pair.SetOrder(mul(9, volumeBuy), volumeBuy)
	idMostLower := pair.SetOrder(mul(10, volumeBuy), volumeBuy)

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
						order := pair.SellLowerOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.SellLowerOrders()[len(pair.SellLowerOrders())-1]
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
						order := pair.SellLowerOrders()[indexSellLower]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.SellLowerOrders()[0]
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
					order := pair.SellLowerOrders()[0]
					if idLower != order.id {
						t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
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
						order := pair.BuyHigherOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
					t.Run("lowest", func(t *testing.T) {
						order := pair.BuyHigherOrders()[len(pair.BuyHigherOrders())-1]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
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
						order := pair.BuyHigherOrders()[indexBuyLower]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.BuyHigherOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
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
						order := pair.BuyHigherOrders()[indexBuyLower]
						if idMostLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idMostLower, order.id)
						}
					})
					t.Run("low", func(t *testing.T) {
						order := pair.BuyHigherOrders()[0]
						if idLower != order.id {
							t.Errorf("id last sell order from array want %v, got %v", idLower, order.id)
						}
					})
				})
			})
		})
	})
}
