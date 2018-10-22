package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
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
	Stakes           []Stake       `json:"stakes"`
	CreatedAtBlock   uint          `json:"created_at_block"`
	Status           byte          `json:"status"`
}

type Validator struct {
	AccumReward string    `json:"accumulated_reward"`
	AbsentTimes int       `json:"absent_times"`
	Candidate   Candidate `json:"candidate"`
}

func makeResponseValidator(v state.Validator, state *state.StateDB) Validator {
	return Validator{
		AccumReward: v.AccumReward.String(),
		AbsentTimes: v.CountAbsentTimes(),
		Candidate:   makeResponseCandidate(*state.GetStateCandidate(v.PubKey)),
	}
}

func makeResponseCandidate(c state.Candidate) Candidate {

	stakes := make([]Stake, len(c.Stakes))

	for i, stake := range c.Stakes {
		stakes[i] = Stake{
			Owner:    stake.Owner,
			Coin:     stake.Coin,
			Value:    stake.Value.String(),
			BipValue: stake.BipValue.String(),
		}
	}

	return Candidate{
		CandidateAddress: c.CandidateAddress,
		TotalStake:       c.TotalBipStake.String(),
		PubKey:           fmt.Sprintf("Mp%x", c.PubKey),
		Commission:       c.Commission,
		Stakes:           stakes,
		CreatedAtBlock:   c.CreatedAtBlock,
		Status:           c.Status,
	}
}

func GetValidators(w http.ResponseWriter, r *http.Request) {

	height, _ := strconv.Atoi(r.URL.Query().Get("height"))

	if height <= 0 {
		height = int(blockchain.Height())
	}

	rState := GetStateForRequest(r)
	vals := rState.GetStateValidators()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if vals == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(Response{
			Code: 404,
			Log:  "Validators not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	var responseValidators []Validator

	for _, val := range vals.Data() {
		responseValidators = append(responseValidators, makeResponseValidator(val, rState))
	}

	_ = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: responseValidators,
	})
}
