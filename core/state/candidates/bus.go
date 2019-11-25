package candidates

import (
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
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

func (b *Bus) GetStakes(pubkey types.Pubkey) []bus.Stake {
	stakes := b.candidates.GetStakes(pubkey)
	var result []bus.Stake

	for _, stake := range stakes {
		result = append(result, bus.Stake{
			Owner:    stake.Owner,
			Value:    big.NewInt(0).Set(stake.Value),
			Coin:     stake.Coin,
			BipValue: big.NewInt(0).Set(stake.BipValue),
		})
	}

	return result
}

func (b *Bus) Punish(height uint64, address types.TmAddress) *big.Int {
	return b.candidates.Punish(height, address)
}

func (b *Bus) GetCandidate(pubkey types.Pubkey) *bus.Candidate {
	candidate := b.candidates.GetCandidate(pubkey)
	if candidate == nil {
		return nil
	}

	return &bus.Candidate{
		PubKey:        candidate.PubKey,
		RewardAddress: candidate.RewardAddress,
		OwnerAddress:  candidate.OwnerAddress,
		Commission:    candidate.Commission,
		Status:        candidate.Status,
	}
}
