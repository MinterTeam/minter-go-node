package swap

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
)

type TradeType int

const (
	TradeTypeExactInput  TradeType = 0
	TradeTypeExactOutput TradeType = 1
)

type Trade struct {
	Route        Route
	TradeType    TradeType
	InputAmount  TokenAmount
	OutputAmount TokenAmount
}

func NewTrade(route Route, amount TokenAmount, tradeType TradeType) *Trade {
	inputAmount, outputAmount := amount, amount
	if tradeType == TradeTypeExactInput {
		for i := 0; i < len(route.Path)-1; i++ {
			pair := route.Pairs[i]
			if pair.Coin1() == amount.Token {
				pair = pair.Reverse()
			}
			tokenAmount, _ := pair.CalculateBuyForSellWithOrders(amount.Amount)
			if tokenAmount == nil {
				return nil
			}

			amount = TokenAmount{Token: pair.Coin1(), Amount: tokenAmount}
		}
		outputAmount = amount
	} else {
		for i := len(route.Path) - 1; i > 0; i-- {
			pair := route.Pairs[i-1]
			if pair.Coin0() == amount.Token {
				pair = pair.Reverse()
			}
			tokenAmount, _ := pair.CalculateSellForBuyWithOrders(amount.Amount)
			if tokenAmount == nil {
				return nil
			}

			amount = TokenAmount{Token: pair.Coin0(), Amount: tokenAmount}
		}
		inputAmount = amount
	}

	if inputAmount.Amount.Sign() < 1 || outputAmount.Amount.Sign() < 1 {
		return nil
	}

	return &Trade{
		Route:        route,
		TradeType:    tradeType,
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
	}
}

func inputOutputComparator(tradeA, tradeB *Trade) int {
	if tradeA.OutputAmount.GetAmount().Cmp(tradeB.OutputAmount.GetAmount()) == 0 {
		// trade A requires less input than trade B, so A should come first
		return tradeA.InputAmount.GetAmount().Cmp(tradeB.InputAmount.GetAmount())
	} else {
		// tradeA has less output than trade B, so should come second
		return tradeA.OutputAmount.GetAmount().Cmp(tradeB.OutputAmount.GetAmount()) * -1
	}
}

func tradeComparator(tradeA, tradeB *Trade) bool {
	ioComp := inputOutputComparator(tradeA, tradeB)
	if ioComp != 0 {
		return ioComp == 1
	}

	// finally consider the number of hops since each hop costs gas
	if len(tradeA.Route.Path) > len(tradeB.Route.Path) {
		return false
	}

	return true
}
func GetBestTradeExactIn(ctx context.Context, pairs []EditableChecker, currencyOut types.CoinID, currencyAmountIn TokenAmount, maxHops int32) *Trade {
	return getBestTradeExactIn(ctx, pairs, currencyOut, currencyAmountIn, maxHops, nil, currencyAmountIn, nil)
}

func getBestTradeExactIn(ctx context.Context, pairs []EditableChecker, currencyOut types.CoinID, currencyAmountIn TokenAmount, maxHops int32, currentPairs []EditableChecker, originalAmountIn TokenAmount, bestTrade *Trade) *Trade {
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

			if bestTrade == nil || tradeComparator(bestTrade, trade) {
				bestTrade = trade
			}
		} else if maxHops > 1 && len(pairs) > 1 { // otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			//pairsExcludingThisPair := append(pairs[:i:i], pairs[i+1:]...)
			temp := make([]EditableChecker, len(pairs))
			copy(temp, pairs)
			temp[i] = temp[len(temp)-1]
			pairsExcludingThisPair := temp[:len(temp)-1]

			newCurrentPairs := append(currentPairs, pair)

			bestTrade = getBestTradeExactIn(ctx, pairsExcludingThisPair, currencyOut, TokenAmount{Amount: amountOut, Token: pair.Coin1()}, maxHops-1, newCurrentPairs, originalAmountIn, bestTrade)
		}
	}

	return bestTrade
}

func GetBestTradeExactOut(ctx context.Context, pairs []EditableChecker, currencyIn types.CoinID, amountOut TokenAmount, maxHops int32) *Trade {
	return getBestTradeExactOut(ctx, pairs, currencyIn, amountOut, maxHops, nil, amountOut, nil)
}

func getBestTradeExactOut(ctx context.Context, pairs []EditableChecker, currencyIn types.CoinID, currencyAmountOut TokenAmount, maxHops int32, currentPairs []EditableChecker, originalAmountOut TokenAmount, bestTrade *Trade) *Trade {
	if maxHops <= 0 {
		return bestTrade
	}

	tokenIn, amountOut, currencyOut := currencyIn, currencyAmountOut, originalAmountOut.GetCurrency()

	for i, pair := range pairs {

		select {
		case <-ctx.Done():
			return bestTrade
		default:
		}

		if pair.Coin0() != amountOut.Token && pair.Coin1() != amountOut.Token {
			continue
		}

		if amountOut.Token == pair.Coin0() {
			pair = pair.Reverse()
		}
		amountIn, _ := pair.CalculateSellForBuyWithOrders(amountOut.Amount)
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

			if bestTrade == nil || tradeComparator(bestTrade, trade) {
				bestTrade = trade
			}
		} else if maxHops > 1 && len(pairs) > 1 { // otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			//pairsExcludingThisPair := append(pairs[:i:i], pairs[i+1:]...)
			temp := make([]EditableChecker, len(pairs))
			copy(temp, pairs)
			temp[i] = temp[len(temp)-1]
			pairsExcludingThisPair := temp[:len(temp)-1]

			newCurrentPairs := append([]EditableChecker{pair}, currentPairs...)

			bestTrade = getBestTradeExactOut(ctx, pairsExcludingThisPair, currencyIn, TokenAmount{Amount: amountIn, Token: pair.Coin0()}, maxHops-1, newCurrentPairs, originalAmountOut, bestTrade)
		}
	}

	return bestTrade
}
