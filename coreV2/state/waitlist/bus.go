package waitlist

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Bus struct {
	waitlist *WaitList
}

func (b *Bus) AddToWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	b.waitlist.AddWaitList(address, pubkey, coin, value)
}

func NewBus(waitlist *WaitList) *Bus {
	return &Bus{waitlist: waitlist}
}
