package swap

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/state/checker"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func init() {
	rand.Seed(1)
}

func TestSwap_GetBestTrade(t *testing.T) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := NewV2(newBus, immutableTree.GetLastImmutable())

	for i := types.CoinID(0); i < 10; i++ {
		for j := i + 1; j < 50; j++ {
			swap.PairCreate(i, j, big.NewInt(rand.Int63n(math.MaxInt64-1)+1), big.NewInt(rand.Int63n(math.MaxInt64-1)+1))
		}
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		t.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}

	swap = NewV2(newBus, immutableTree.GetLastImmutable())
	swap.SwapPools(context.Background())

	swap.trader = &traderV1{}
	tradeOut := swap.GetBestTradeExactIn(context.Background(), 0, 1, big.NewInt(1e18), 4)
	tradeIn := swap.GetBestTradeExactOut(context.Background(), 1, 0, big.NewInt(7933854205489585136), 4)

	swap.trader = &traderV2{}
	t.Run("GetBestTradeExactInV2", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			trade := swap.GetBestTradeExactIn(context.Background(), 0, 1, big.NewInt(1e18), 4)
			if trade.OutputAmount.Amount.Cmp(tradeOut.OutputAmount.Amount) == -1 {
				t.Error(trade.Route.Path, tradeOut.Route.Path)
				t.Fatal(trade.OutputAmount.Amount, tradeOut.OutputAmount.Amount)
			}
		}

	})
	t.Run("GetBestTradeExactOutV2", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			trade := swap.GetBestTradeExactOut(context.Background(), 1, 0, big.NewInt(7933854205489585136), 4)
			if trade.InputAmount.Amount.Cmp(tradeIn.InputAmount.Amount) == 1 {
				t.Error(trade.Route.Path, tradeIn.Route.Path)
				t.Fatal(trade.InputAmount.Amount, tradeIn.InputAmount.Amount)
			}
		}
	})

}
func BenchmarkSwap_GetBestTrade(b *testing.B) {
	memDB := db.NewMemDB()
	immutableTree, err := tree.NewMutableTree(0, memDB, 1024, 0)
	if err != nil {
		b.Fatal(err)
	}
	newBus := bus.NewBus()
	checker.NewChecker(newBus)
	swap := NewV2(newBus, immutableTree.GetLastImmutable())

	for i := types.CoinID(0); i < 20; i++ {
		for j := i + 1; j < 30; j++ {
			swap.PairCreate(i, j, big.NewInt(rand.Int63n(math.MaxInt64-1)+1), big.NewInt(rand.Int63n(math.MaxInt64-1)+1))
		}
	}

	_, _, err = immutableTree.Commit(swap)
	if err != nil {
		b.Fatal(err)
	}
	immutableTree, err = tree.NewMutableTree(1, memDB, 1024, 0)
	if err != nil {
		b.Fatal(err)
	}

	swap = NewV2(newBus, immutableTree.GetLastImmutable())
	swap.SwapPools(context.Background())

	b.Run("GetBestTradeExactInV2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			swap.GetBestTradeExactIn(context.Background(), 0, 1, big.NewInt(1e18), 4)
		}
	})
	b.Run("GetBestTradeExactOutV2", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			swap.GetBestTradeExactOut(context.Background(), 1, 0, big.NewInt(7933854205489585136), 4)
		}
	})

	swap.trader = &traderV1{}
	b.Run("GetBestTradeExactInV1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			swap.GetBestTradeExactIn(context.Background(), 0, 1, big.NewInt(1e18), 4)
		}
	})
	b.Run("GetBestTradeExactOutV1", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			swap.GetBestTradeExactOut(context.Background(), 1, 0, big.NewInt(7933854205489585136), 4)
		}
	})
}
