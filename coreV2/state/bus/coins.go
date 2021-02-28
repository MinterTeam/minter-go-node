package bus

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Coins interface {
	// Deprecated
	GetCoinV1(types.CoinID) *Coin

	GetCoin(types.CoinID) *Coin
	SubCoinVolume(types.CoinID, *big.Int)
	SubCoinReserve(types.CoinID, *big.Int)
}

type Coin struct {
	ID      types.CoinID
	Name    string
	Crr     uint32
	Symbol  types.CoinSymbol
	Version types.CoinVersion
	Volume  *big.Int
	Reserve *big.Int
}

func (m Coin) GetFullSymbol() string {
	if m.Version == 0 {
		return m.Symbol.String()
	}

	return fmt.Sprintf("%s-%d", m.Symbol, m.Version)
}
