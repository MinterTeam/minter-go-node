package candidates

import (
	"github.com/MinterTeam/minter-go-node/coreV2/state/bus"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
)

type Bus struct {
	candidates *Candidates
}

func NewBus(candidates *Candidates) *Bus {
	return &Bus{candidates: candidates}
}

// GetStakes returns list of stakes of candidate with given public key
func (b *Bus) GetStakes(pubkey types.Pubkey) []*bus.Stake {
	stakes := b.candidates.GetStakes(pubkey)
	var result []*bus.Stake

	for _, stake := range stakes {
		result = append(result, &bus.Stake{
			Owner:    stake.Owner,
			Value:    big.NewInt(0).Set(stake.Value),
			Coin:     stake.Coin,
			BipValue: big.NewInt(0).Set(stake.BipValue),
		})
	}

	return result
}

// Punish punished a candidate with given tendermint-address
func (b *Bus) Punish(height uint64, address types.TmAddress) {
	b.candidates.Punish(height, address)
}

// ID returns id by a public key
func (b *Bus) ID(pubkey types.Pubkey) uint32 {
	return b.candidates.ID(pubkey)
}

// GetCandidate returns candidate by a public key
func (b *Bus) GetCandidate(pubkey types.Pubkey) *bus.Candidate {
	candidate := b.candidates.GetCandidate(pubkey)
	if candidate == nil {
		return nil
	}

	return &bus.Candidate{
		ID:             candidate.ID,
		PubKey:         candidate.PubKey,
		RewardAddress:  candidate.RewardAddress,
		OwnerAddress:   candidate.OwnerAddress,
		ControlAddress: candidate.ControlAddress,
		Commission:     candidate.Commission,
		Status:         candidate.Status,
		AddUpdate: func(id types.CoinID, value, bipValue *big.Int, address types.Address) {
			candidate.addUpdate(&stake{
				Owner:    address,
				Coin:     id,
				Value:    big.NewInt(0).Set(value),
				BipValue: big.NewInt(0).Set(bipValue),
			})
			return
		},
	}
}

// SetOffline sets candidate status to CandidateStatusOffline
func (b *Bus) SetOffline(pubkey types.Pubkey) {
	b.candidates.SetOffline(pubkey)
}

func (b *Bus) TotalStakes() *big.Int {
	return b.candidates.TotalStakes()
}

// GetCandidateByTendermintAddress finds and returns candidate with given tendermint-address
func (b *Bus) GetCandidateByTendermintAddress(tmAddress types.TmAddress) *bus.Candidate {
	candidate := b.candidates.GetCandidateByTendermintAddress(tmAddress)
	if candidate == nil {
		return nil
	}

	return &bus.Candidate{
		ID:             candidate.ID,
		PubKey:         candidate.PubKey,
		RewardAddress:  candidate.RewardAddress,
		OwnerAddress:   candidate.OwnerAddress,
		ControlAddress: candidate.ControlAddress,
		Commission:     candidate.Commission,
		Status:         candidate.Status,
	}
}
