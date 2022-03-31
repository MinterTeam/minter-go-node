package swap

import "math/big"

type TokenAmount struct {
	Token  Token
	Amount *big.Int
}

type Token struct {
	CoinID uint64
}

func NewToken(coinID uint64) Token {
	return Token{CoinID: coinID}
}

func (t Token) IsEqual(other Token) bool {
	return t.CoinID == other.CoinID
}

func NewTokenAmount(token Token, amount *big.Int) TokenAmount {
	return TokenAmount{Token: token, Amount: amount}
}

func (ta TokenAmount) GetAmount() *big.Int {
	return ta.Amount
}

func (ta TokenAmount) GetCurrency() Token {
	return ta.Token
}

func (ta TokenAmount) add(other TokenAmount) TokenAmount {
	return TokenAmount{
		Token:  ta.Token,
		Amount: new(big.Int).Add(ta.Amount, other.Amount),
	}
}

func (ta TokenAmount) sub(other TokenAmount) TokenAmount {
	return TokenAmount{
		Token:  ta.Token,
		Amount: new(big.Int).Sub(ta.Amount, other.Amount),
	}
}
