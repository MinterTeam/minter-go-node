package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Accounts interface {
	DeleteCoin(types.Address, types.CoinSymbol)
	AddBalance(types.Address, types.CoinSymbol, *big.Int)
}
