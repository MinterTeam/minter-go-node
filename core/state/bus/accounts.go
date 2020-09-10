package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Accounts interface {
	AddBalance(types.Address, types.CoinID, *big.Int)
}
