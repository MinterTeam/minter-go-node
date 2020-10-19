package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
)

type Stake struct {
	Owner    string `json:"owner"`
	Coin     Coin   `json:"coin"`
	Value    string `json:"value"`
	BipValue string `json:"bip_value"`
}

type CandidateResponse struct {
	RewardAddress string  `json:"reward_address"`
	OwnerAddress  string  `json:"owner_address"`
	TotalStake    string  `json:"total_stake"`
	PubKey        string  `json:"pub_key"`
	Commission    uint32  `json:"commission"`
	Stakes        []Stake `json:"stakes,omitempty"`
	Status        byte    `json:"status"`
}

func makeResponseCandidate(state *state.CheckState, c candidates.Candidate, includeStakes bool) CandidateResponse {
	candidate := CandidateResponse{
		RewardAddress: c.RewardAddress.String(),
		OwnerAddress:  c.OwnerAddress.String(),
		TotalStake:    state.Candidates().GetTotalStake(c.PubKey).String(),
		PubKey:        c.PubKey.String(),
		Commission:    c.Commission,
		Status:        c.Status,
	}

	if includeStakes {
		stakes := state.Candidates().GetStakes(c.PubKey)
		candidate.Stakes = make([]Stake, len(stakes))
		for i, stake := range stakes {
			candidate.Stakes[i] = Stake{
				Owner: stake.Owner.String(),
				Coin: Coin{
					ID:     stake.Coin.Uint32(),
					Symbol: state.Coins().GetCoin(stake.Coin).GetFullSymbol(),
				},
				Value:    stake.Value.String(),
				BipValue: stake.BipValue.String(),
			}
		}
	}

	return candidate
}

func Candidate(pubkey types.Pubkey, height int) (*CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	if height != 0 {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakesOfCandidate(pubkey)
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidate := cState.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Candidate not found"}
	}

	response := makeResponseCandidate(cState, *candidate, true)
	return &response, nil
}
