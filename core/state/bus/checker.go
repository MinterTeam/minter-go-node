package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Checker interface {
	AddCoin(types.CoinSymbol, *big.Int)
}
