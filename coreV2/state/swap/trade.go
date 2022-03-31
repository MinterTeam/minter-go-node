package swap

import (
	"math/big"
	"sort"
)

type TradeType int

const (
	TradeTypeExactInput  TradeType = 0
	TradeTypeExactOutput TradeType = 1
)

type Trade struct {
	Route          Route
	TradeType      TradeType
	InputAmount    TokenAmount
	OutputAmount   TokenAmount
	ExecutionPrice Price
	NextMidPrice   Price
	PriceImpact    *big.Int
	Profit         *big.Int
}

type TradeCollection []*Trade

func (t TradeCollection) Len() int           { return len(t) }
func (t TradeCollection) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t TradeCollection) Less(i, j int) bool { return tradeComparator(t[i], t[j]) }

func NewTrade(route Route, amount TokenAmount, tradeType TradeType) (*Trade, error) {
	amounts := make([]TokenAmount, len(route.Path))
	nextPairs := make([]*PairTrade, len(route.Pairs))

	var inputAmount, outputAmount TokenAmount
	if tradeType == TradeTypeExactInput {
		amounts[0] = amount
		for i := 0; i < len(route.Path)-1; i++ {
			tokenAmount, nextPair, err := route.Pairs[i].GetOutputAmount(amounts[i])
			if err != nil {
				return nil, err
			}

			amounts[i+1], nextPairs[i] = tokenAmount, nextPair
		}

		inputAmount, outputAmount = amount, amounts[len(amounts)-1]
	} else {
		amounts[len(amounts)-1] = amount
		for i := len(route.Path) - 1; i > 0; i-- {
			tokenAmount, nextPair, err := route.Pairs[i-1].GetInputAmount(amounts[i])
			if err != nil {
				return nil, err
			}

			amounts[i-1], nextPairs[i-1] = tokenAmount, nextPair
		}

		outputAmount, inputAmount = amount, amounts[0]
	}

	if inputAmount.Amount.Cmp(big.NewInt(0)) == 0 || outputAmount.Amount.Cmp(big.NewInt(0)) == 0 {
		return nil, ErrInsufficientReserve
	}

	return &Trade{
		Route:          route,
		TradeType:      tradeType,
		InputAmount:    inputAmount,
		OutputAmount:   outputAmount,
		ExecutionPrice: NewPrice(inputAmount.GetCurrency(), outputAmount.GetCurrency(), inputAmount.GetAmount(), outputAmount.GetAmount()),
		NextMidPrice:   NewPriceFromRoute(NewRoute(nextPairs, route.Input, nil)),
		Profit:         big.NewInt(0).Sub(outputAmount.GetAmount(), inputAmount.GetAmount()),
	}, nil
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
		if tradeA.InputAmount.GetAmount().Cmp(tradeB.InputAmount.GetAmount()) == 0 {
			return 0
		}

		// trade A requires less input than trade B, so A should come first
		if tradeA.InputAmount.GetAmount().Cmp(tradeB.InputAmount.GetAmount()) == -1 {
			return -1
		} else {
			return 1
		}
	} else {
		// tradeA has less output than trade B, so should come second
		if tradeA.OutputAmount.GetAmount().Cmp(tradeB.OutputAmount.GetAmount()) == -1 {
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

	// consider lowest slippage next, since these are less likely to fail
	if tradeA.PriceImpact.Cmp(tradeB.PriceImpact) == -1 {
		return true
	} else if tradeA.PriceImpact.Cmp(tradeB.PriceImpact) == 1 {
		return false
	}

	// finally consider the number of hops since each hop costs gas
	if len(tradeA.Route.Path) > len(tradeB.Route.Path) {
		return false
	}

	return true
}

func GetBestTradeExactIn(pairs []*PairTrade, currencyOut Token, currencyAmountIn TokenAmount, options TradeOptions) ([]*Trade, error) {
	if options.MaxHops <= 0 {
		return nil, nil
	}

	return getBestTradeExactIn(pairs, currencyOut, currencyAmountIn, options, make([]*PairTrade, 0), currencyAmountIn)
}

func getBestTradeExactIn(
	pairs []*PairTrade,
	currencyOut Token,
	currencyAmountIn TokenAmount,
	tradeOptions TradeOptions,
	currentPairs []*PairTrade,
	originalAmountIn TokenAmount,
	bestTrades ...*Trade,
) ([]*Trade, error) {
	if tradeOptions.MaxHops <= 0 {
		return bestTrades, nil
	}

	tokenOut, tokenAmountIn := currencyOut, currencyAmountIn

	for i, pair := range pairs {
		if !pair.Token0.Token.IsEqual(tokenAmountIn.Token) && !pair.Token1.Token.IsEqual(tokenAmountIn.Token) {
			continue
		}

		if pair.getReserve0().Sign() == 0 || pair.getReserve1().Sign() == 0 {
			continue
		}

		amountOut, _, err := pair.GetOutputAmount(tokenAmountIn)
		if err != nil {
			if err == ErrInsufficientReserve {
				continue
			}
			return bestTrades, err
		}

		// we have arrived at the output token, so this is the final trade of one of the paths
		if amountOut.Token.IsEqual(tokenOut) {
			trade, err := NewTrade(
				NewRoute(append(currentPairs, pair), originalAmountIn.GetCurrency(), &currencyOut),
				originalAmountIn,
				TradeTypeExactInput,
			)

			if err != nil {
				continue
			}

			bestTrades = append(bestTrades, trade)
			sort.Sort(sort.Reverse(TradeCollection(bestTrades)))

			if len(bestTrades) > tradeOptions.MaxNumResults {
				bestTrades = (bestTrades)[:tradeOptions.MaxNumResults]
			}
		} else if tradeOptions.MaxHops > 1 && len(pairs) > 1 {
			// otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			pairsExcludingThisPair := append(append([]*PairTrade{}, pairs[:i]...), pairs[i+1:]...)
			newCurrentPairs := append(currentPairs, pair)
			newTradeOptions := TradeOptions{tradeOptions.MaxNumResults, tradeOptions.MaxHops - 1}

			var err error
			bestTrades, err = getBestTradeExactIn(
				pairsExcludingThisPair,
				currencyOut,
				amountOut,
				newTradeOptions,
				newCurrentPairs,
				originalAmountIn,
				bestTrades...,
			)

			if err != nil {
				return nil, err
			}
		}
	}

	return bestTrades, nil
}

func GetBestTradeExactOut(pairs []*PairTrade, currencyIn Token, amountOut TokenAmount, options TradeOptions) ([]*Trade, error) {
	if options.MaxHops <= 0 {
		return nil, nil
	}

	return getBestTradeExactOut(pairs, currencyIn, amountOut, options, make([]*PairTrade, 0), amountOut)
}

func getBestTradeExactOut(
	pairs []*PairTrade,
	currencyIn Token,
	currencyAmountOut TokenAmount,
	tradeOptions TradeOptions,
	currentPairs []*PairTrade,
	originalAmountOut TokenAmount,
	bestTrades ...*Trade,
) ([]*Trade, error) {
	if tradeOptions.MaxHops <= 0 {
		return bestTrades, nil
	}

	tokenIn, amountOut, currencyOut := currencyIn, currencyAmountOut, originalAmountOut.GetCurrency()

	for i, pair := range pairs {
		if !pair.Token0.Token.IsEqual(amountOut.Token) && !pair.Token1.Token.IsEqual(amountOut.Token) {
			continue
		}

		if pair.getReserve0().Cmp(big.NewInt(0)) == 0 || pair.getReserve1().Cmp(big.NewInt(0)) == 0 {
			continue
		}

		amountIn, _, err := pair.GetInputAmount(amountOut)
		if err != nil {
			if err == ErrInsufficientReserve {
				continue
			}

			return nil, err
		}

		if amountIn.Token.IsEqual(tokenIn) {
			trade, err := NewTrade(
				NewRoute(append([]*PairTrade{pair}, currentPairs...), currencyIn, &currencyOut),
				originalAmountOut,
				TradeTypeExactOutput,
			)

			if err != nil {
				continue
			}

			bestTrades = append(bestTrades, trade)
			sort.Sort(sort.Reverse(TradeCollection(bestTrades)))

			if len(bestTrades) > tradeOptions.MaxNumResults {
				bestTrades = (bestTrades)[:tradeOptions.MaxNumResults]
			}
		} else if tradeOptions.MaxHops > 1 && len(pairs) > 1 {
			// otherwise, consider all the other paths that lead from this token as long as we have not exceeded maxHops
			pairsExcludingThisPair := append(append([]*PairTrade{}, pairs[:i]...), pairs[i+1:]...)
			newCurrentPairs := append([]*PairTrade{pair}, currentPairs...)
			newTradeOptions := TradeOptions{tradeOptions.MaxNumResults, tradeOptions.MaxHops - 1}

			var err error
			bestTrades, err = getBestTradeExactOut(
				pairsExcludingThisPair,
				currencyIn,
				amountIn,
				newTradeOptions,
				newCurrentPairs,
				originalAmountOut,
				bestTrades...,
			)

			if err != nil {
				return nil, err
			}
		}
	}

	return bestTrades, nil
}
