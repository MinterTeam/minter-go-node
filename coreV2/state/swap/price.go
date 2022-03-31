package swap

import (
	"math/big"
)

type Price struct {
	BaseToken  Token
	QuoteToken Token
	Value      *big.Int
}

func NewPrice(baseToken Token, quoteToken Token, denominator *big.Int, numerator *big.Int) Price {
	wei, value := new(big.Int), new(big.Float).Quo(new(big.Float).SetInt(numerator), new(big.Float).SetInt(denominator))
	new(big.Float).Mul(value, big.NewFloat(1e18)).Int(wei)

	return Price{
		BaseToken:  baseToken,
		QuoteToken: quoteToken,
		Value:      wei,
	}
}

func NewPriceFromRoute(route Route) Price {
	var prices []Price
	for i, pair := range route.Pairs {
		if route.Path[i].IsEqual(pair.Token0.Token) {
			prices = append(prices, NewPrice(pair.Token0.Token, pair.Token1.Token, pair.getReserve0(), pair.getReserve1()))
		} else {
			prices = append(prices, NewPrice(pair.Token1.Token, pair.Token0.Token, pair.getReserve1(), pair.getReserve0()))
		}
	}

	result := big.NewInt(0)
	for i, price := range prices {
		if i == 0 {
			continue
		}

		result = result.Mul(result, price.Value)
	}

	result = result.Add(result, prices[0].Value)

	return Price{
		BaseToken:  prices[0].BaseToken,
		QuoteToken: prices[0].QuoteToken,
		Value:      result,
	}
}
