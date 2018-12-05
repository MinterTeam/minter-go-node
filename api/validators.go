package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/pkg/errors"
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
	PubKey           types.Pubkey  `json:"pub_key"`
	Commission       uint          `json:"commission"`
	Stakes           []Stake       `json:"stakes,omitempty"`
	CreatedAtBlock   uint          `json:"created_at_block"`
	Status           byte          `json:"status"`
}

type ValidatorResponse struct {
	AccumReward *big.Int          `json:"accumulated_reward"`
	AbsentTimes int               `json:"absent_times"`
	Candidate   CandidateResponse `json:"candidate"`
}

func makeResponseValidator(v state.Validator, state *state.StateDB) ValidatorResponse {
	return ValidatorResponse{
		AccumReward: v.AccumReward,
		AbsentTimes: v.CountAbsentTimes(),
		Candidate:   makeResponseCandidate(*state.GetStateCandidate(v.PubKey), false),
	}
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

func Validators(height int) (*ResponseValidators, error) {
	rState, err := GetStateForHeight(height)

	if err != nil {
		return nil, err
	}

	vals := rState.GetStateValidators()
	if vals == nil {
		return nil, errors.New("Validator not found")
	}

	var responseValidators ResponseValidators
	for _, val := range vals.Data() {
		responseValidators = append(responseValidators, makeResponseValidator(val, rState))
	}

	return &responseValidators, nil
}
