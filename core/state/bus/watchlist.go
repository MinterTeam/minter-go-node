package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type WatchList interface {
	AddToWatchList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int)
}
