package swap

import (
	"errors"
	"math/big"
)

var ErrInsufficientReserve = errors.New("insufficient reserve")

type PairTrade struct {
	Token0 TokenAmount
	Token1 TokenAmount
}

func NewPair(tokenAmountA TokenAmount, tokenAmountB TokenAmount) *PairTrade {
	return &PairTrade{
		Token0: tokenAmountA,
		Token1: tokenAmountB,
	}
}

func (p PairTrade) GetOutputAmount(inputAmount TokenAmount) (TokenAmount, *PairTrade, error) {
	if p.getReserve0().Cmp(big.NewInt(0)) == 0 || p.getReserve1().Cmp(big.NewInt(0)) == 0 {
		return TokenAmount{}, nil, ErrInsufficientReserve
	}

	inputReserve := p.getReserveOf(inputAmount.Token)
	outputReserve := p.Token0
	if p.Token0.Token.IsEqual(inputAmount.Token) {
		outputReserve = p.Token1
	}

	inputAmountWithFee := new(big.Int).Mul(inputAmount.Amount, big.NewInt(998))
	numerator := new(big.Int).Mul(inputAmountWithFee, outputReserve.Amount)
	denominator := new(big.Int).Add(new(big.Int).Mul(inputReserve.Amount, big.NewInt(1000)), inputAmountWithFee)

	outputAmount := TokenAmount{
		Token:  outputReserve.Token,
		Amount: new(big.Int).Div(numerator, denominator),
	}

	return outputAmount, NewPair(inputReserve.add(inputAmount), outputReserve.sub(outputAmount)), nil
}

func (p PairTrade) GetInputAmount(outputAmount TokenAmount) (TokenAmount, *PairTrade, error) {
	if p.getReserve0().Cmp(big.NewInt(0)) == 0 || p.getReserve1().Cmp(big.NewInt(0)) == 0 || p.getReserveOf(outputAmount.Token).Amount.Cmp(outputAmount.Amount) == -1 {
		return TokenAmount{}, nil, ErrInsufficientReserve
	}

	outputReserve := p.getReserveOf(outputAmount.Token)
	inputReserve := p.Token0
	if p.Token0.Token.IsEqual(outputAmount.Token) {
		inputReserve = p.Token1
	}

	numerator := new(big.Int).Mul(new(big.Int).Mul(inputReserve.Amount, outputAmount.Amount), big.NewInt(1000))
	denominator := new(big.Int).Mul(new(big.Int).Sub(outputReserve.Amount, outputAmount.Amount), big.NewInt(998))

	amount := big.NewInt(0)
	if denominator.Cmp(amount) != 0 {
		amount = new(big.Int).Add(new(big.Int).Div(numerator, denominator), big.NewInt(1))
	}

	inputAmount := TokenAmount{
		Token:  inputReserve.Token,
		Amount: amount,
	}

	return inputAmount, NewPair(inputReserve.add(inputAmount), outputReserve.sub(outputAmount)), nil
}

func (p PairTrade) getReserve0() *big.Int {
	return p.Token0.Amount
}

func (p PairTrade) getReserve1() *big.Int {
	return p.Token1.Amount
}

func (p PairTrade) getReserveOf(token Token) TokenAmount {
	if p.Token0.Token.IsEqual(token) {
		return p.Token0
	}

	return p.Token1
}
