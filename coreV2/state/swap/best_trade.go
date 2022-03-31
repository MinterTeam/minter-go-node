package swap

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

func (s *Swap) GetBestTradeExactOut(ctx context.Context, inId, outId uint64, outAmount *big.Int, maxHops uint64) *Trade {
	s.loadPools()

	var pairs []EditableChecker

	s.muPairs.RLock()
	for _, pair := range s.pairs {
		if pair == nil {
			continue
		}
		reserve0, reserve1 := pair.Reserves()
		if reserve0.Sign() < 1 || reserve1.Sign() < 1 {
			continue
		}

		pairs = append(pairs, pair)
	}
	s.muPairs.RUnlock()

	trade := GetBestTradeExactOut(ctx,
		pairs,
		types.CoinID(inId),
		NewTokenAmount(types.CoinID(outId), outAmount),
		maxHops,
	)

	return trade
}
func (s *SwapV2) GetBestTradeExactOut(ctx context.Context, inId, outId uint64, outAmount *big.Int, maxHops uint64) *Trade {
	s.loadPools()

	var pairs []EditableChecker

	s.muPairs.RLock()
	for _, pair := range s.pairs {
		if pair == nil {
			continue
		}
		reserve0, reserve1 := pair.Reserves()
		if reserve0.Sign() < 1 || reserve1.Sign() < 1 {
			continue
		}

		pairs = append(pairs, pair)
	}
	s.muPairs.RUnlock()

	trade := GetBestTradeExactOut(ctx, pairs, types.CoinID(inId), NewTokenAmount(types.CoinID(outId), outAmount), maxHops)

	return trade
}
func (s *Swap) GetBestTradeExactIn(ctx context.Context, outId, inId uint64, inAmount *big.Int, maxHops uint64) *Trade {
	s.loadPools()

	var pairs []EditableChecker

	s.muPairs.RLock()
	for _, pair := range s.pairs {
		if pair == nil {
			continue
		}
		reserve0, reserve1 := pair.Reserves()
		if reserve0.Sign() < 1 || reserve1.Sign() < 1 {
			continue
		}
		pairs = append(pairs, pair)
	}
	s.muPairs.RUnlock()

	trades := GetBestTradeExactIn(ctx,
		pairs,
		types.CoinID(outId),
		NewTokenAmount(types.CoinID(inId), inAmount),
		maxHops,
	)

	return trades
}
func (s *SwapV2) GetBestTradeExactIn(ctx context.Context, outId, inId uint64, inAmount *big.Int, maxHops uint64) *Trade {
	s.loadPools()

	var pairs []EditableChecker

	s.muPairs.RLock()
	for _, pair := range s.pairs {
		if pair == nil {
			continue
		}
		reserve0, reserve1 := pair.Reserves()
		if reserve0.Sign() < 1 || reserve1.Sign() < 1 {
			continue
		}
		pairs = append(pairs, pair)
	}
	s.muPairs.RUnlock()

	trades := GetBestTradeExactIn(ctx, pairs, types.CoinID(outId), NewTokenAmount(types.CoinID(inId), inAmount), maxHops)

	return trades
}
