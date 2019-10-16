package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Coins interface {
	GetCoin(types.CoinSymbol) *Coin
	SubCoinVolume(types.CoinSymbol, *big.Int)
	SubCoinReserve(types.CoinSymbol, *big.Int)
	SanitizeCoin(types.CoinSymbol)

	AddOwner(types.CoinSymbol, types.Address)
	RemoveOwner(types.CoinSymbol, types.Address)
}

type Coin struct {
	Name    string
	Crr     uint
	Symbol  types.CoinSymbol
	Volume  *big.Int
	Reserve *big.Int
}
