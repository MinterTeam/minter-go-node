package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type Bus struct {
	candidates *Candidates
}

func NewBus(candidates *Candidates) *Bus {
	return &Bus{candidates: candidates}
}

func (b *Bus) DeleteCoin(pubkey types.Pubkey, coin types.CoinSymbol) {
	b.candidates.DeleteCoin(pubkey, coin)
}
