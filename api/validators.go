package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Stake struct {
	Owner    types.Address    `json:"owner"`
	Coin     types.CoinSymbol `json:"coin"`
	Value    string           `json:"value"`
	BipValue string           `json:"bip_value"`
}

type CandidateResponse struct {
	CandidateAddress types.Address `json:"candidate_address"`
	TotalStake       *big.Int      `json:"total_stake"`
	PubKey           types.Pubkey  `json:"pubkey"`
	Commission       uint          `json:"commission"`
	Stakes           []Stake       `json:"stakes,omitempty"`
	CreatedAtBlock   uint          `json:"created_at_block"`
	Status           byte          `json:"status"`
}

type ValidatorResponse struct {
	Pubkey types.Pubkey `json:"pubkey"`
}

func makeResponseCandidate(c state.Candidate, includeStakes bool) CandidateResponse {
	candidate := CandidateResponse{
		CandidateAddress: c.CandidateAddress,
		TotalStake:       c.TotalBipStake,
		PubKey:           c.PubKey,
		Commission:       c.Commission,
		CreatedAtBlock:   c.CreatedAtBlock,
		Status:           c.Status,
	}

	if includeStakes {
		candidate.Stakes = make([]Stake, len(c.Stakes))
		for i, stake := range c.Stakes {
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

type ResponseValidators []ValidatorResponse

func Validators(height int64) (*ResponseValidators, error) {
	tmVals, err := client.Validators(&height)
	if err != nil {
		return nil, err
	}

	responseValidators := make(ResponseValidators, len(tmVals.Validators))
	for i, val := range tmVals.Validators {
		responseValidators[i] = ValidatorResponse{
			Pubkey: val.PubKey.Bytes()[5:],
		}
	}

	return &responseValidators, nil
}
