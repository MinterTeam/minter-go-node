package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type GetCoin struct {
	Symbol types.CoinSymbol
}

type Coin struct {
	Name    string
	Crr     uint
	Symbol  types.CoinSymbol
	Volume  *big.Int
	Reserve *big.Int
}

type SubCoinVolume struct {
	Symbol types.CoinSymbol
	Amount *big.Int
}

type SubCoinReserve struct {
	Symbol types.CoinSymbol
	Amount *big.Int
}

type SanitizeCoin struct {
	Symbol types.CoinSymbol
}
