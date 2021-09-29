package waitlist

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Bus struct {
	waitlist *WaitList
}

func (b *Bus) AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	b.waitlist.AddWaitList(address, pubkey, coin, value)
}
func (b *Bus) Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID) {
	b.waitlist.Delete(address, pubkey, coin)
}
func (b *Bus) GetByAddressAndPubKey(address types.Address, pubkey types.Pubkey) (res []*bus.WaitListItem) {
	waitlist := b.waitlist.GetByAddressAndPubKey(address, pubkey)
	for _, item := range waitlist {
		res = append(res, &bus.WaitListItem{
			CandidateId: item.CandidateId,
			Coin:        item.Coin,
			Value:       big.NewInt(0).Set(item.Value),
		})
	}
	return res
}

func NewBus(waitlist *WaitList) *Bus {
	return &Bus{waitlist: waitlist}
}
