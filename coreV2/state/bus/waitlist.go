package bus

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type WaitList interface {
	AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int)
}
