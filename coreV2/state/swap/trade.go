package swap

type TradeType int

const (
	TradeTypeExactInput  TradeType = 0
	TradeTypeExactOutput TradeType = 1
)

type Trade struct {
	Route        Route
	TradeType    TradeType
	InputAmount  *TokenAmount
	OutputAmount *TokenAmount
}

func NewTrade(route Route, amount *TokenAmount, tradeType TradeType) *Trade {
	inputAmount, outputAmount := amount, amount
	if tradeType == TradeTypeExactInput {
		for i := 0; i < len(route.Path)-1; i++ {
			pair := route.Pairs[i]

			if pair.Coin1() == outputAmount.Token {
				pair = pair.Reverse()
			}
			tokenAmount, _ := pair.CalculateBuyForSellWithOrders(outputAmount.Amount)
			if tokenAmount == nil {
				return nil
			}


			outputAmount = &TokenAmount{Token: pair.Coin1(), Amount: tokenAmount}
		}
	} else {
		for i := len(route.Path) - 1; i > 0; i-- {
			pair := route.Pairs[i-1]
			if pair.Coin0() == inputAmount.Token {
				pair = pair.Reverse()
			}
			tokenAmount, _ := pair.CalculateSellForBuyWithOrders(inputAmount.Amount)
			if tokenAmount == nil {
				return nil
			}
			inputAmount = &TokenAmount{Token: pair.Coin0(), Amount: tokenAmount}
		}
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
