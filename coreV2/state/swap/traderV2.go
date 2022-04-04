package swap

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type traderV2 struct {
}

func (t *traderV2) GetBestTradeExactIn(ctx context.Context, pairs []EditableChecker, currencyOut types.CoinID, currencyAmountIn *TokenAmount, maxHops int32) *Trade {
	return t.getBestTradeExactIn(ctx, pairs, currencyOut, currencyAmountIn, maxHops, nil, currencyAmountIn, nil, make(map[types.CoinID]*bestTokenAmount, 2000))
}

func (t *traderV2) getBestTradeExactIn(ctx context.Context, pairs []EditableChecker, currencyOut types.CoinID, currencyAmountIn *TokenAmount, maxHops int32, currentPairs []EditableChecker, originalAmountIn *TokenAmount, bestTrade *Trade, checks map[types.CoinID]*bestTokenAmount) *Trade {
	if maxHops <= 0 {
		return bestTrade
	}

	tokenOut, tokenAmountIn := currencyOut, currencyAmountIn

	for i, pair := range pairs {

		select {
		case <-ctx.Done():
			return bestTrade
		default:
		}

		if maxHops == 1 && pair.Coin0() != tokenOut && pair.Coin1() != tokenOut {
			continue
		}

		if pair.Coin0() != tokenAmountIn.Token && pair.Coin1() != tokenAmountIn.Token {
			continue
		}

		if tokenAmountIn.Token == pair.Coin1() {
			pair = pair.Reverse()
		}
		amountOut, _ := pair.CalculateBuyForSellWithOrders(tokenAmountIn.Amount)
		if amountOut == nil {
			continue
		}

		if pair.Coin1() == tokenOut { // we have arrived at the output token, so this is the final trade of one of the paths
			trade := NewTrade(
				NewRoute(append(currentPairs, pair), originalAmountIn.GetCurrency(), &currencyOut),
				originalAmountIn,
				TradeTypeExactInput,
			)

			if trade == nil {
				continue
			}

			if maxPrevTokenRoute, ok := checks[pair.Coin0()]; !ok || (ok && maxPrevTokenRoute.lastHop <= maxHops && maxPrevTokenRoute.amount.Cmp(tokenAmountIn.Amount) == -1) { // todo: || maxHops
				newMax := &bestTokenAmount{
					amount:  tokenAmountIn.Amount,
					lastHop: maxHops + 1,
				}
				checks[pair.Coin0()] = newMax
			}

			if bestTrade == nil || tradeComparator(bestTrade, trade) {
				bestTrade = trade
			}
		} else if maxHops > 1 && len(pairs) > 1 { // otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			if maxPrevTokenRoute, ok := checks[pair.Coin1()]; ok && maxPrevTokenRoute.lastHop >= maxHops && maxPrevTokenRoute.amount.Cmp(amountOut) == 1 { // todo: || maxHops
				continue
			}

			otherPair := append(pairs[:i:i], pairs[i+1:]...)
			newCurrentPairs := append(currentPairs, pair)

			bestTrade = t.getBestTradeExactIn(ctx, otherPair, currencyOut, NewTokenAmount(pair.Coin1(), amountOut), maxHops-1, newCurrentPairs, originalAmountIn, bestTrade, checks)
		}
	}

	return bestTrade
}

type bestTokenAmount struct {
	amount  *big.Int
	lastHop int32
}

func (t *traderV2) GetBestTradeExactOut(ctx context.Context, pairs []EditableChecker, currencyIn types.CoinID, amountOut *TokenAmount, maxHops int32) *Trade {
	return t.getBestTradeExactOut(ctx, pairs, currencyIn, amountOut, maxHops, nil, amountOut, nil, make(map[types.CoinID]*bestTokenAmount, 2000))
}

func (t *traderV2) getBestTradeExactOut(ctx context.Context, pairs []EditableChecker, currencyIn types.CoinID, currencyAmountOut *TokenAmount, maxHops int32, currentPairs []EditableChecker, originalAmountOut *TokenAmount, bestTrade *Trade, checks map[types.CoinID]*bestTokenAmount) *Trade {
	if maxHops <= 0 {
		return bestTrade
	}

	tokenIn, tokenAmountOut, currencyOut := currencyIn, currencyAmountOut, originalAmountOut.GetCurrency()

	for i, pair := range pairs {

		select {
		case <-ctx.Done():
			return bestTrade
		default:
		}

		if maxHops == 1 && pair.Coin0() != tokenIn && pair.Coin1() != tokenIn {
			continue
		}

		if pair.Coin0() != tokenAmountOut.Token && pair.Coin1() != tokenAmountOut.Token {
			continue
		}

		if tokenAmountOut.Token == pair.Coin0() {
			pair = pair.Reverse()
		}
		amountIn, _ := pair.CalculateSellForBuyWithOrders(tokenAmountOut.Amount)
		if amountIn == nil {
			continue
		}

		if pair.Coin0() == tokenIn {
			trade := NewTrade(
				NewRoute(append([]EditableChecker{pair}, currentPairs...), currencyIn, &currencyOut),
				originalAmountOut,
				TradeTypeExactOutput,
			)

			if trade == nil {
				continue
			}

			if maxPrevTokenRoute, ok := checks[pair.Coin1()]; !ok || (ok && maxPrevTokenRoute.lastHop <= maxHops && maxPrevTokenRoute.amount.Cmp(tokenAmountOut.Amount) == 1) { // todo: || maxHops
				checks[pair.Coin1()] = &bestTokenAmount{
					amount:  tokenAmountOut.Amount,
					lastHop: maxHops + 1,
				}
			}

			if bestTrade == nil || tradeComparator(bestTrade, trade) {
				bestTrade = trade
			}
		} else if maxHops > 1 && len(pairs) > 1 { // otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			if maxPrevTokenRoute, ok := checks[pair.Coin0()]; ok && maxPrevTokenRoute.lastHop >= maxHops && maxPrevTokenRoute.amount.Cmp(amountIn) == -1 { // todo: || maxHops
				continue
			}

			otherPair := append(pairs[:i:i], pairs[i+1:]...)
			newCurrentPairs := append([]EditableChecker{pair}, currentPairs...)

			bestTrade = t.getBestTradeExactOut(ctx, otherPair, currencyIn, NewTokenAmount(pair.Coin0(), amountIn), maxHops-1, newCurrentPairs, originalAmountOut, bestTrade, checks)
		}
	}

	return bestTrade
}
