package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Bus struct {
	waitlist *WaitList
}

func (b *Bus) AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int, lock bus.WaitlistItemLock) {
	var l *lockedValue
	if lock != nil {
		l = &lockedValue{
			ToHeight: lock.GetToHeight(),
			Value:    lock.GetValue(),
			From:     lock.GetFrom(),
		}
	}
	b.waitlist.AddWaitList(address, pubkey, coin, value, l)
}

func (b *Bus) Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) bus.WaitlistItem {
	return b.waitlist.Get(address, pubkey, coin)
}

func (b *Bus) Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID) {
	b.waitlist.Delete(address, pubkey, coin)
}
func (b *Bus) GetFree(address types.Address, pubkey types.Pubkey, coin types.CoinID, height uint64) *big.Int {
	return b.waitlist.GetFree(address, pubkey, coin, height)
}

func NewBus(waitlist *WaitList) *Bus {
	return &Bus{waitlist: waitlist}
}
