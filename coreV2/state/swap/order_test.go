package swap

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"strconv"
	"testing"
)

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
						if pair.OrderBuyHigherByIndex(0).Price().Cmp(pair.OrderBuyHigherByIndex(len(pair.BuyHigherOrders())-1).Price()) != -1 {
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
						if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(len(pair.SellLowerOrders())-1).Price()) != -1 {
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
						if pair.OrderSellLowerByIndex(0).Price().Cmp(pair.OrderSellLowerByIndex(indexSellLower).Price()) != -1 {
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

func TestPair_CalculateBuyForSellWithOrders(t *testing.T) {
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
		t.Run("sell", func(t *testing.T) {
			price := big.NewFloat(0.5)
			amount0 := pair.CalculateSellAmount0ForPrice(price)
			if amount0.Cmp(big.NewInt(4147)) != 0 {
				t.Fatalf("amount0 want %v, got %v", big.NewInt(4147), amount0)
			}
			amount1 := pair.CalculateBuyForSell(amount0)
			p := pair.AddLastSwapStep(amount0, amount1)
			if p.Price().Text('f', 18) != "0.499964656817699860" {
				t.Error(amount0, amount1, p.Price(), price)
			}
		})
		t.Run("buy", func(t *testing.T) {
			price := big.NewFloat(0.5)
			amount1 := pair.CalculateBuyAmount1ForPrice(price)
			if amount1.Cmp(big.NewInt(2927)) != 0 {
				t.Fatalf("amount1 want %v, got %v", big.NewInt(2927), amount1)
			}
			amount0 := pair.CalculateSellForBuy(amount1)
			p := pair.AddLastSwapStep(amount0, amount1)
			if p.Price().Text('f', 18) != "0.499964656817699860" {
				t.Error(amount0, amount1, p.Price(), price)
			}
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
			amount0In := pair.CalculateSellForBuy(big.NewInt(5000))
			amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(5000))
			if amount0In.Cmp(amount0InWithOB) != 0 {
				t.Error("not equal", amount0In, amount0InWithOB)
			}
		})
	})

	t.Run("with orders", func(t *testing.T) {
		t.Run("one order", func(t *testing.T) {
			pair.SetOrder(big.NewInt(2000), big.NewInt(1000))
			if pair.OrderSellLowerByIndex(0).Price().Cmp(calcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
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
				t.Run("one order", func(t *testing.T) {
					t.Run("before first order", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuy(big.NewInt(2927))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927))
						if amount0In.Cmp(amount0InWithOB) != 0 {
							t.Error("not equal", amount0In, amount0InWithOB)
						}
					})
					t.Run("first order", func(t *testing.T) {
						amount0In := pair.CalculateSellForBuy(big.NewInt(2927))
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
						amount0 := pair.CalculateSellForBuy(big.NewInt(2927))
						p := pair.AddLastSwapStep(amount0, big.NewInt(2927))
						amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 999 + 466))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "2000" {
							t.Error("want to get 1,000-0.1% more and spend 2,000 more by order", amount0In, amount0InWithOB)
						}
					})
				})
			})
			t.Run("two orders", func(t *testing.T) {
				pair.SetOrder(big.NewInt(2000), big.NewInt(1000))
				if pair.OrderSellLowerByIndex(0).Price().Cmp(calcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 &&
					pair.OrderSellLowerByIndex(1).Price().Cmp(calcPriceSell(big.NewInt(2000), big.NewInt(1000))) != 0 {
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
						amount0 := pair.CalculateSellForBuy(big.NewInt(2927))
						p := pair.AddLastSwapStep(amount0, big.NewInt(2927))
						amount0In := big.NewInt(0).Add(amount0, p.CalculateSellForBuy(big.NewInt(466)))
						amount0InWithOB := pair.CalculateSellForBuyWithOrders(big.NewInt(2927 + 1998 + 466))
						if big.NewInt(0).Sub(amount0InWithOB, amount0In).String() != "4000" {
							t.Error("want to get 2,000-0.1% more and spend 4,000 more by order", amount0In, amount0InWithOB)
						}
					})
				})
			})
		})
	})
}
