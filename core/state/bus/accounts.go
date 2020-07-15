package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Accounts interface {
	DeleteCoin(types.Address, types.CoinID)
	AddBalance(types.Address, types.CoinID, *big.Int)
}
