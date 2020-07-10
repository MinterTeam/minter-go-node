package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Coins interface {
	GetCoin(types.CoinID) *Coin
	SubCoinVolume(types.CoinID, *big.Int)
	SubCoinReserve(types.CoinID, *big.Int)
}

type Coin struct {
	ID      types.CoinID
	Name    string
	Crr     uint
	Symbol  types.CoinSymbol
	Volume  *big.Int
	Reserve *big.Int
}
