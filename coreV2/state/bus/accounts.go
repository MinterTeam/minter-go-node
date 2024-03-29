package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Accounts interface {
	AddBalance(types.Address, types.CoinID, *big.Int)
	IsX3Mining(addr types.Address, height uint64) bool
	GetLockStakeUntilBlock(address types.Address) (height uint64)
	GetBalance(address types.Address, coin types.CoinID) *big.Int
}
