package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type WatchList interface {
	AddFrozenFund(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int)
}
