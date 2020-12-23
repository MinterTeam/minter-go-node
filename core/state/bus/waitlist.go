package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type WaitlistItem interface {
	GetFree(height uint64) *big.Int
	GetAll() *big.Int
}

type WaitList interface {
	AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int, height uint64)
	Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) WaitlistItem
	Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID)
	GetFree(address types.Address, pubkey types.Pubkey, coin types.CoinID, height uint64) *big.Int
}
