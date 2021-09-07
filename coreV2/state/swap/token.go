package swap

type Token struct {
	CoinID uint64
}

func NewToken(coinID uint64) Token {
	return Token{CoinID: coinID}
}

func (t Token) IsEqual(other Token) bool {
	return t.CoinID == other.CoinID
}
