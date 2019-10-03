package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
)

type Stake struct {
	Owner    types.Address    `json:"owner"`
	Coin     types.CoinSymbol `json:"coin"`
	Value    string           `json:"value"`
	BipValue string           `json:"bip_value"`
}

type CandidateResponse struct {
	RewardAddress types.Address `json:"reward_address"`
	OwnerAddress  types.Address `json:"owner_address"`
	TotalStake    *big.Int      `json:"total_stake"`
	PubKey        types.Pubkey  `json:"pub_key"`
	Commission    uint          `json:"commission"`
	Stakes        []Stake       `json:"stakes,omitempty"`
	Status        byte          `json:"status"`
}

func makeResponseCandidate(state *state.State, c candidates.Candidate, includeStakes bool) CandidateResponse {
	candidate := CandidateResponse{
		RewardAddress: c.RewardAddress,
		OwnerAddress:  c.OwnerAddress,
		TotalStake:    state.Candidates.GetTotalStake(c.PubKey),
		PubKey:        c.PubKey,
		Commission:    c.Commission,
		Status:        c.Status,
	}

	if includeStakes {
		stakes := state.Candidates.GetStakes(c.PubKey)
		candidate.Stakes = make([]Stake, len(stakes))
		for i, stake := range stakes {
			candidate.Stakes[i] = Stake{
				Owner:    stake.Owner,
				Coin:     stake.Coin,
				Value:    stake.Value.String(),
				BipValue: stake.BipValue.String(),
			}
		}
	}

	return candidate
}

func Candidate(pubkey []byte, height int) (*CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	candidate := cState.Candidates.GetCandidate(pubkey)
	if candidate == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Candidate not found"}
	}

	response := makeResponseCandidate(cState, *candidate, true)
	return &response, nil
}
