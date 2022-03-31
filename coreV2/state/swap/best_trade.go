package swap

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

func (s *Swap) GetBestTradeExactOut(ctx context.Context, inId, outId uint64, outAmount *big.Int, maxHops uint64) *Trade {
	trade := GetBestTradeExactOut(ctx,
		s.SwapPools(),
		types.CoinID(inId),
		NewTokenAmount(types.CoinID(outId), outAmount),
		maxHops,
	)

	return trade
}
func (s *SwapV2) GetBestTradeExactOut(ctx context.Context, inId, outId uint64, outAmount *big.Int, maxHops uint64) *Trade {
	trade := GetBestTradeExactOut(ctx, s.SwapPools(), types.CoinID(inId), NewTokenAmount(types.CoinID(outId), outAmount), maxHops)

	return trade
}
func (s *Swap) GetBestTradeExactIn(ctx context.Context, outId, inId uint64, inAmount *big.Int, maxHops uint64) *Trade {
	trades := GetBestTradeExactIn(ctx,
		s.SwapPools(),
		types.CoinID(outId),
		NewTokenAmount(types.CoinID(inId), inAmount),
		maxHops,
	)

	return trades
}
func (s *SwapV2) GetBestTradeExactIn(ctx context.Context, outId, inId uint64, inAmount *big.Int, maxHops uint64) *Trade {

	trades := GetBestTradeExactIn(ctx, s.SwapPools(), types.CoinID(outId), NewTokenAmount(types.CoinID(inId), inAmount), maxHops)

	return trades
}
