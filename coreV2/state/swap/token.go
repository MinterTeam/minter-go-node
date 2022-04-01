package swap

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type TokenAmount struct {
	Token  types.CoinID
	Amount *big.Int
}

type Token struct {
	CoinID uint64
}

func NewToken(coinID uint64) Token {
	return Token{CoinID: coinID}
}

func NewTokenAmount(token types.CoinID, amount *big.Int) TokenAmount {
	return TokenAmount{Token: token, Amount: amount}
}

func (ta TokenAmount) GetAmount() *big.Int {
	return ta.Amount
}

func (ta TokenAmount) GetCurrency() types.CoinID {
	return ta.Token
}
