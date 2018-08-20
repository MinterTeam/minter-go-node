package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"net/http"
	"strconv"
)

type Stake struct {
	Owner    types.Address    `json:"owner"`
	Coin     types.CoinSymbol `json:"coin"`
	Value    string           `json:"value"`
	BipValue string           `json:"bip_value"`
}

type Candidate struct {
	CandidateAddress types.Address `json:"candidate_address"`
	TotalStake       string        `json:"total_stake"`
	PubKey           string        `json:"pub_key"`
	Commission       uint          `json:"commission"`
	AccumReward      string        `json:"accumulated_reward"`
	Stakes           []Stake       `json:"stakes"`
	CreatedAtBlock   uint          `json:"created_at_block"`
	Status           byte          `json:"status"`
	AbsentTimes      uint          `json:"absent_times"`
}

func makeResponseCandidate(c state.Candidate, state *state.StateDB) Candidate {

	stakes := make([]Stake, len(c.Stakes))

	for i, stake := range c.Stakes {
		stakes[i] = Stake{
			Owner:    stake.Owner,
			Coin:     stake.Coin,
			Value:    stake.Value.String(),
			BipValue: stake.BipValue(state).String(),
		}
	}

	return Candidate{
		CandidateAddress: c.CandidateAddress,
		TotalStake:       c.TotalBipStake.String(),
		PubKey:           fmt.Sprintf("Mp%x", c.PubKey),
		Commission:       c.Commission,
		AccumReward:      c.AccumReward.String(),
		Stakes:           stakes,
		CreatedAtBlock:   c.CreatedAtBlock,
		Status:           c.Status,
		AbsentTimes:      c.AbsentTimes,
	}
}

func GetValidators(w http.ResponseWriter, r *http.Request) {

	height, _ := strconv.Atoi(r.URL.Query().Get("height"))

	if height <= 0 {
		height = int(blockchain.Height())
	}

	rState := GetStateForRequest(r)
	_, candidates := rState.GetValidators(validators.GetValidatorsCountForBlock(uint64(height)))

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	var responseCandidates []Candidate

	for _, candidate := range candidates {
		responseCandidates = append(responseCandidates, makeResponseCandidate(candidate, rState))
	}

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: responseCandidates,
	})
}
