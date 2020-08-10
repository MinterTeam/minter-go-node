package watchlist

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Bus struct {
	watchlist *WatchList
}

func (b *Bus) AddFrozenFund(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	b.watchlist.AddWatchList(address, pubkey, coin, value)
}

func NewBus(watchlist *WatchList) *Bus {
	return &Bus{watchlist: watchlist}
}
