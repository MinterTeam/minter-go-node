package swap

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
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
	Profit       *big.Int
}

func NewTrade(route Route, amount TokenAmount, tradeType TradeType) *Trade {
	var amountNew TokenAmount

	var inputAmount, outputAmount TokenAmount
	if tradeType == TradeTypeExactInput {
		amountNew = amount
		for i := 0; i < len(route.Path)-1; i++ {
			pair := route.Pairs[i]
			tokenAmount, _ := pair.CalculateBuyForSellWithOrders(amountNew.Amount)
			if tokenAmount == nil {
				return nil
			}

			amountNew = TokenAmount{Token: pair.Coin1(), Amount: tokenAmount}
		}

		inputAmount, outputAmount = amount, amountNew
	} else {
		amountNew = amount
		for i := len(route.Path) - 1; i > 0; i-- {
			pair := route.Pairs[i-1]
			tokenAmount, _ := pair.CalculateSellForBuyWithOrders(amountNew.Amount)
			if tokenAmount == nil {
				return nil
			}

			amountNew = TokenAmount{Token: pair.Coin0(), Amount: tokenAmount}
		}

		outputAmount, inputAmount = amount, amountNew
	}

	if inputAmount.Amount.Sign() < 1 || outputAmount.Amount.Sign() < 1 {
		return nil
	}

	return &Trade{
		Route:        route,
		TradeType:    tradeType,
		InputAmount:  inputAmount,
		OutputAmount: outputAmount,
		Profit:       big.NewInt(0).Sub(outputAmount.GetAmount(), inputAmount.GetAmount()),
	}
}

func (t *Trade) GetMaximumAmountIn(slippageTolerance float64) TokenAmount {
	if t.TradeType == TradeTypeExactInput {
		return t.InputAmount
	}

	maximumAmountIn := new(big.Int)
	inputAmount := new(big.Float).SetInt(t.InputAmount.GetAmount())
	percent := big.NewFloat(1 + slippageTolerance)
	new(big.Float).Mul(inputAmount, percent).Int(maximumAmountIn)

	return NewTokenAmount(t.InputAmount.Token, maximumAmountIn)
}

func (t *Trade) GetMinimumAmountOut(slippageTolerance float64) TokenAmount {
	if t.TradeType == TradeTypeExactOutput {
		return t.OutputAmount
	}

	minimumAmountOut := new(big.Int)
	outputAmount := new(big.Float).SetInt(t.OutputAmount.GetAmount())
	percent := big.NewFloat(1 + slippageTolerance)
	new(big.Float).Quo(outputAmount, percent).Int(minimumAmountOut)

	return NewTokenAmount(t.InputAmount.Token, minimumAmountOut)
}

type TradeOptions struct {
	MaxNumResults int
	MaxHops       int
}

func inputOutputComparator(tradeA, tradeB *Trade) int {
	if tradeA.OutputAmount.GetAmount().Cmp(tradeB.OutputAmount.GetAmount()) == 0 {
		if tradeA.InputAmount.GetAmount() == tradeB.InputAmount.GetAmount() {
			return 0
		}

		// trade A requires less input than trade B, so A should come first
		if tradeA.InputAmount.GetAmount().Cmp(tradeB.InputAmount.GetAmount()) < 0 {
			return -1
		} else {
			return 1
		}
	} else {
		// tradeA has less output than trade B, so should come second
		if tradeA.OutputAmount.GetAmount().Cmp(tradeB.OutputAmount.GetAmount()) < 0 {
			return 1
		} else {
			return -1
		}
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
func GetBestTradeExactIn(ctx context.Context, pairs []EditableChecker, currencyOut types.CoinID, currencyAmountIn TokenAmount, maxHops uint64) *Trade {
	if maxHops <= 0 {
		return nil
	}

	return getBestTradeExactIn(ctx, pairs, currencyOut, currencyAmountIn, maxHops, nil, currencyAmountIn, nil)
}

func getBestTradeExactIn(ctx context.Context, pairs []EditableChecker, currencyOut types.CoinID, currencyAmountIn TokenAmount, maxHops uint64, currentPairs []EditableChecker, originalAmountIn TokenAmount, bestTrade *Trade) *Trade {
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

		amountOut, _ := pair.CalculateBuyForSellWithOrders(tokenAmountIn.Amount)
		if amountOut != nil {
			continue
		}

		// we have arrived at the output token, so this is the final trade of one of the paths
		if pair.Coin1() == tokenOut {
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
		} else if maxHops > 1 && len(pairs) > 1 {
			// otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			pairsExcludingThisPair := append(pairs[:i], pairs[i+1:]...)
			newCurrentPairs := append(currentPairs, pair)

			bestTrade = getBestTradeExactIn(ctx, pairsExcludingThisPair, currencyOut, TokenAmount{Amount: amountOut, Token: pair.Coin0()}, maxHops-1, newCurrentPairs, originalAmountIn, bestTrade)
		}
	}

	return bestTrade
}

func GetBestTradeExactOut(ctx context.Context, pairs []EditableChecker, currencyIn types.CoinID, amountOut TokenAmount, maxHops uint64) *Trade {
	if maxHops <= 0 {
		return nil
	}

	return getBestTradeExactOut(ctx, pairs, currencyIn, amountOut, maxHops, nil, amountOut, nil)
}

func getBestTradeExactOut(ctx context.Context, pairs []EditableChecker, currencyIn types.CoinID, currencyAmountOut TokenAmount, maxHops uint64, currentPairs []EditableChecker, originalAmountOut TokenAmount, bestTrade *Trade) *Trade {
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
		} else if maxHops > 1 && len(pairs) > 1 {
			// otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			pairsExcludingThisPair := append(pairs[:i], pairs[i+1:]...)
			newCurrentPairs := append([]EditableChecker{pair}, currentPairs...)

			var err error
			bestTrade = getBestTradeExactOut(ctx, pairsExcludingThisPair, currencyIn, TokenAmount{Amount: amountIn, Token: pair.Coin1()}, maxHops-1, newCurrentPairs, originalAmountOut, bestTrade)

			if err != nil {
				return nil
			}
		}
	}

	return bestTrade
}
