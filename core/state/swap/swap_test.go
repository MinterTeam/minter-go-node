package swap

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/iavl"
	db "github.com/tendermint/tm-db"
	"math/big"
	"reflect"
	"testing"
)

func TestPair_feeToOff(t *testing.T) {
	t.Parallel()
	tableTests := []struct {
		coin0, coin1                     types.CoinID
		token0Amount, token1Amount       *big.Int
		swapAmount, expectedOutputAmount *big.Int
		expectedLiquidity                *big.Int
	}{
		{
			coin0:                0,
			coin1:                1,
			token0Amount:         new(big.Int).Add(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)), big.NewInt(0)),
			token1Amount:         new(big.Int).Add(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)), big.NewInt(0)),
			swapAmount:           new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			expectedOutputAmount: big.NewInt(996006981039903216),
			expectedLiquidity:    new(big.Int).Add(new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)), big.NewInt(0)),
		},
	}

	service := New(nil, iavl.NewImmutableTree(db.NewMemDB(), 0))
	for i, tt := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			err := service.CheckMint(tt.coin0, tt.coin1, tt.token0Amount, tt.token1Amount)
			if err != nil {
				t.Fatal(err)
			}
			pair := service.ReturnPair(tt.coin0, tt.coin1)

			liquidity := pair.Mint(types.Address{1}, tt.token0Amount, tt.token1Amount)

			expectedLiquidity := new(big.Int).Sub(tt.expectedLiquidity, big.NewInt(minimumLiquidity))
			if liquidity.Cmp(expectedLiquidity) != 0 {
				t.Errorf("liquidity want %s, got %s", expectedLiquidity, liquidity)
			}

			_, _ = pair.Swap(big.NewInt(0), tt.swapAmount, tt.expectedOutputAmount, big.NewInt(0))

			err = service.CheckBurn(types.Address{1}, tt.coin0, tt.coin1, expectedLiquidity, big.NewInt(0), big.NewInt(0))
			if err != nil {
				t.Fatal(err)
			}
			_, _ = pair.Burn(types.Address{1}, expectedLiquidity, big.NewInt(0), big.NewInt(0))

			if pair.GetTotalSupply().Cmp(big.NewInt(minimumLiquidity)) != 0 {
				t.Errorf("liquidity want %s, got %s", big.NewInt(minimumLiquidity), pair.GetTotalSupply())
			}
		})
	}
}

func TestPair_Mint(t *testing.T) {
	t.Parallel()
	tableTests := []struct {
		coin0, coin1               types.CoinID
		token0Amount, token1Amount *big.Int
		expectedLiquidity          *big.Int
	}{
		{
			coin0:             0,
			coin1:             1,
			token0Amount:      new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			token1Amount:      new(big.Int).Add(new(big.Int).Mul(big.NewInt(4), big.NewInt(1e18)), big.NewInt(0)),
			expectedLiquidity: new(big.Int).Add(new(big.Int).Mul(big.NewInt(2), big.NewInt(1e18)), big.NewInt(0)),
		},
	}
	service := New(nil, iavl.NewImmutableTree(db.NewMemDB(), 0))
	for i, tt := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			err := service.CheckMint(tt.coin0, tt.coin1, tt.token0Amount, tt.token1Amount)
			if err != nil {
				t.Fatal(err)
			}
			pair := service.ReturnPair(tt.coin0, tt.coin1)

			liquidity := pair.Mint(types.Address{1}, tt.token0Amount, tt.token1Amount)

			liquidityExpected := new(big.Int).Sub(tt.expectedLiquidity, big.NewInt(minimumLiquidity))
			if liquidity.Cmp(liquidityExpected) != 0 {
				t.Errorf("liquidity want %s, got %s", liquidityExpected, liquidity)
			}

			reserve0, reserve1 := pair.Reserve0, pair.Reserve1

			if reserve0.Cmp(tt.token0Amount) != 0 {
				t.Errorf("reserve0 want %s, got %s", tt.token0Amount, reserve0)
			}

			if reserve1.Cmp(tt.token1Amount) != 0 {
				t.Errorf("reserve1 want %s, got %s", tt.token1Amount, reserve1)
			}

			if pair.balances[types.Address{}].Liquidity.Cmp(big.NewInt(minimumLiquidity)) != 0 {
				t.Errorf("addressZero liquidity want %s, got %s", big.NewInt(minimumLiquidity), pair.balances[types.Address{}].Liquidity)
			}

			if pair.GetTotalSupply().Cmp(tt.expectedLiquidity) != 0 {
				t.Errorf("total supply want %s, got %s", big.NewInt(minimumLiquidity), pair.GetTotalSupply())
			}
		})
	}
}

func TestPair_Swap_token0(t *testing.T) {
	t.Parallel()
	tableTests := []struct {
		coin0, coin1               types.CoinID
		token0Amount, token1Amount *big.Int
		swap0Amount                *big.Int
		expected1OutputAmount      *big.Int
	}{
		{
			coin0:                 1,
			coin1:                 2,
			token0Amount:          new(big.Int).Add(new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18)), big.NewInt(0)),
			token1Amount:          new(big.Int).Add(new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)), big.NewInt(0)),
			swap0Amount:           new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			expected1OutputAmount: big.NewInt(1662497915624478906),
		},
	}
	service := New(nil, iavl.NewImmutableTree(db.NewMemDB(), 0))
	for i, tt := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			err := service.CheckMint(tt.coin0, tt.coin1, tt.token0Amount, tt.token1Amount)
			if err != nil {
				t.Fatal(err)
			}
			pair := service.ReturnPair(tt.coin0, tt.coin1)

			_ = pair.Mint(types.Address{1}, tt.token0Amount, tt.token1Amount)

			err = service.CheckSwap(tt.coin0, tt.coin1, tt.swap0Amount, new(big.Int).Add(tt.expected1OutputAmount, big.NewInt(1)))
			if err != ErrorK {
				t.Fatalf("failed with %v, want error %v", err, ErrorK)
			}

			err = service.CheckSwap(tt.coin0, tt.coin1, tt.swap0Amount, tt.expected1OutputAmount)
			if err != nil {
				t.Fatal(err)
			}

			amount0, amount1 := pair.Swap(tt.swap0Amount, big.NewInt(0), big.NewInt(0), tt.expected1OutputAmount)

			if amount0.Cmp(tt.swap0Amount) != 0 {
				t.Errorf("amount0 want %s, got %s", tt.swap0Amount, amount0)
			}

			amount1.Neg(amount1)
			if amount1.Cmp(tt.expected1OutputAmount) != 0 {
				t.Errorf("amount1 want %s, got %s", tt.expected1OutputAmount, amount1)
			}

			if pair.Reserve0.Cmp(new(big.Int).Add(tt.token0Amount, tt.swap0Amount)) != 0 {
				t.Errorf("reserve0 want %s, got %s", new(big.Int).Add(tt.token0Amount, tt.swap0Amount), pair.Reserve0)
			}

			if pair.Reserve1.Cmp(new(big.Int).Sub(tt.token1Amount, tt.expected1OutputAmount)) != 0 {
				t.Errorf("Reserve1 want %s, got %s", new(big.Int).Add(tt.token1Amount, tt.expected1OutputAmount), pair.Reserve1)
			}
		})
	}
}

func TestPair_Swap_token1(t *testing.T) {
	t.Parallel()
	tableTests := []struct {
		coin0, coin1               types.CoinID
		token0Amount, token1Amount *big.Int
		swap1Amount                *big.Int
		expected0OutputAmount      *big.Int
	}{
		{
			coin0:                 1,
			coin1:                 2,
			token0Amount:          new(big.Int).Add(new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18)), big.NewInt(0)),
			token1Amount:          new(big.Int).Add(new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)), big.NewInt(0)),
			swap1Amount:           new(big.Int).Add(new(big.Int).Mul(big.NewInt(1), big.NewInt(1e18)), big.NewInt(0)),
			expected0OutputAmount: big.NewInt(453305446940074565),
		},
	}
	service := New(nil, iavl.NewImmutableTree(db.NewMemDB(), 0))
	for i, tt := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			err := service.CheckMint(tt.coin0, tt.coin1, tt.token0Amount, tt.token1Amount)
			if err != nil {
				t.Fatal(err)
			}
			pair := service.ReturnPair(tt.coin0, tt.coin1)

			_ = pair.Mint(types.Address{1}, tt.token0Amount, tt.token1Amount)

			err = service.CheckSwap(tt.coin1, tt.coin0, tt.swap1Amount, new(big.Int).Add(tt.expected0OutputAmount, big.NewInt(1)))
			if err != ErrorK {
				t.Fatalf("failed with %v, want error %v", err, ErrorK)
			}

			err = service.CheckSwap(tt.coin1, tt.coin0, tt.swap1Amount, tt.expected0OutputAmount)
			if err != nil {
				t.Fatal(err)
			}

			amount0, amount1 := pair.Swap(big.NewInt(0), tt.swap1Amount, tt.expected0OutputAmount, big.NewInt(0))

			amount0.Neg(amount0)
			if amount0.Cmp(tt.expected0OutputAmount) != 0 {
				t.Errorf("amount0 want %s, got %s", tt.expected0OutputAmount, amount0)
			}

			if amount1.Cmp(tt.swap1Amount) != 0 {
				t.Errorf("amount1 want %s, got %s", tt.swap1Amount, amount1)
			}

			if pair.Reserve0.Cmp(new(big.Int).Sub(tt.token0Amount, tt.expected0OutputAmount)) != 0 {
				t.Errorf("reserve0 want %s, got %s", new(big.Int).Sub(tt.token0Amount, tt.expected0OutputAmount), pair.Reserve0)
			}

			if pair.Reserve1.Cmp(new(big.Int).Add(tt.token1Amount, tt.swap1Amount)) != 0 {
				t.Errorf("reserve1 want %s, got %s", new(big.Int).Add(tt.token1Amount, tt.swap1Amount), pair.Reserve1)
			}
		})
	}
}

func TestPair_Burn(t *testing.T) {
	t.Parallel()
	tableTests := []struct {
		coin0, coin1               types.CoinID
		token0Amount, token1Amount *big.Int
		expectedLiquidity          *big.Int
	}{
		{
			coin0:             0,
			coin1:             1,
			token0Amount:      new(big.Int).Add(new(big.Int).Mul(big.NewInt(3), big.NewInt(1e18)), big.NewInt(0)),
			token1Amount:      new(big.Int).Add(new(big.Int).Mul(big.NewInt(3), big.NewInt(1e18)), big.NewInt(0)),
			expectedLiquidity: new(big.Int).Add(new(big.Int).Mul(big.NewInt(3), big.NewInt(1e18)), big.NewInt(0)),
		},
	}
	service := New(nil, iavl.NewImmutableTree(db.NewMemDB(), 0))
	for i, tt := range tableTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			err := service.CheckMint(tt.coin0, tt.coin1, tt.token0Amount, tt.token1Amount)
			if err != nil {
				t.Fatal(err)
			}
			pair := service.ReturnPair(tt.coin0, tt.coin1)

			liquidity := pair.Mint(types.Address{1}, tt.token0Amount, tt.token1Amount)

			liquidityExpected := new(big.Int).Sub(tt.expectedLiquidity, big.NewInt(minimumLiquidity))
			if liquidity.Cmp(liquidityExpected) != 0 {
				t.Errorf("liquidity want %s, got %s", liquidityExpected, liquidity)
			}

			err = service.CheckBurn(types.Address{1}, tt.coin0, tt.coin1, liquidity, big.NewInt(0), big.NewInt(0))
			if err != nil {
				t.Fatal(err)
			}

			amount0, amount1 := pair.Burn(types.Address{1}, liquidity, big.NewInt(0), big.NewInt(0))

			expectedAmount0 := new(big.Int).Sub(tt.token0Amount, big.NewInt(minimumLiquidity))
			if amount0.Cmp(expectedAmount0) != 0 {
				t.Errorf("amount0 want %s, got %s", expectedAmount0, amount0)
			}

			expectedAmount1 := new(big.Int).Sub(tt.token1Amount, big.NewInt(minimumLiquidity))
			if amount1.Cmp(expectedAmount1) != 0 {
				t.Errorf("amount1 want %s, got %s", expectedAmount1, amount1)
			}

			if pair.balances[types.Address{1}].Liquidity.Sign() != 0 {
				t.Errorf("address liquidity want %s, got %s", "0", pair.balances[types.Address{1}].Liquidity)
			}

			if pair.balances[types.Address{}].Liquidity.Cmp(big.NewInt(minimumLiquidity)) != 0 {
				t.Errorf("addressZero liquidity want %s, got %s", big.NewInt(minimumLiquidity), pair.balances[types.Address{}].Liquidity)
			}

			if pair.GetTotalSupply().Cmp(big.NewInt(minimumLiquidity)) != 0 {
				t.Errorf("total supply want %s, got %s", big.NewInt(minimumLiquidity), pair.GetTotalSupply())
			}
		})
	}
}

func TestSwap_Pair_reverseKey(t *testing.T) {
	t.Parallel()
	service := New(nil, iavl.NewImmutableTree(db.NewMemDB(), 0))
	pair := service.Pair(0, 1)
	if pair != nil {
		t.Fatal("pair is not nil")
	}
	pair = service.ReturnPair(0, 1)
	if pair == nil {
		t.Fatal("pair is nil")
	}
	pair = service.Pair(0, 1)
	if pair == nil {
		t.Fatal("pair is nil")
	}
	address := types.Address{1}
	err := service.CheckMint(0, 1, big.NewInt(1e18), big.NewInt(2e18))
	if err != nil {
		t.Fatal(err)
	}
	liquidity := pair.Mint(address, big.NewInt(1e18), big.NewInt(2e18))
	if liquidity == nil {
		t.Error("liquidity is nil")
	}
	if !reflect.DeepEqual(liquidity, pair.Balance(address)) {
		t.Error("liquidities is equal")
	}
	reserve0, reserve1 := pair.Reserves()
	totalSupply := pair.GetTotalSupply()

	pairReverted := service.Pair(1, 0)
	if pairReverted == nil {
		t.Fatal("pairReverted is nil")
	}
	reserve0Reverted, reserve1Reverted := pairReverted.Reserves()
	totalSupplyReverted := pairReverted.GetTotalSupply()

	if reserve0.Cmp(reserve1Reverted) != 0 {
		t.Error(reserve0, reserve1Reverted)
	}
	if reserve1.Cmp(reserve0Reverted) != 0 {
		t.Error(reserve1, reserve0Reverted)
	}
	if totalSupply.Cmp(totalSupplyReverted) != 0 {
		t.Error(totalSupply, totalSupplyReverted)
	}
	if !reflect.DeepEqual(pair.balances, pairReverted.balances) {
		t.Error("balances not equal")
	}

	if pairReverted.isDirty != pair.isDirty {
		t.Error("isDirty not equal")
	}
	pair.isDirty = !pair.isDirty
	if pairReverted.isDirty != pair.isDirty {
		t.Error("isDirty not equal")
	}
}
