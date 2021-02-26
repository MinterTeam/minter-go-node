package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type FrozenFunds interface {
	AddFrozenFund(uint64, types.Address, types.Pubkey, uint32, types.CoinID, *big.Int)
}
