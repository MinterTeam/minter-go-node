package bus

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type WaitlistItem interface {
	GetFree(height uint64) *big.Int
	GetAll() *big.Int
	GetLocked() []WaitlistItemLock
}

type WaitlistItemLock interface {
	GetToHeight() uint64
	GetValue() *big.Int
	GetFrom() uint32
}

type WaitList interface {
	AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int, height WaitlistItemLock)
	Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) WaitlistItem
	Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID)
	GetFree(address types.Address, pubkey types.Pubkey, coin types.CoinID, height uint64) *big.Int
}
