package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Accounts interface {
	AddBalance(types.Address, types.CoinID, *big.Int)
}
