package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"net/http"
)

func GetCandidate(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pubkey := types.Hex2Bytes(vars["pubkey"])

	candidate := blockchain.CurrentState().GetStateCandidate(pubkey)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: struct {
			CandidateAddress types.Address `json:"candidate_address"`
			TotalStake       string        `json:"total_stake"`
			PubKey           string        `json:"pub_key"`
			Commission       uint          `json:"commission"`
			AccumReward      string        `json:"accumulated_reward"`
			Stakes           []state.Stake `json:"stakes"`
			CreatedAtBlock   uint          `json:"created_at_block"`
			Status           byte          `json:"status"`
			AbsentTimes      uint          `json:"absent_times"`
		}{
			CandidateAddress: candidate.CandidateAddress,
			TotalStake:       candidate.TotalStake.String(),
			PubKey:           fmt.Sprintf("Mx%x", candidate.PubKey),
			Commission:       candidate.Commission,
			AccumReward:      candidate.AccumReward.String(),
			Stakes:           candidate.Stakes,
			CreatedAtBlock:   candidate.CreatedAtBlock,
			Status:           candidate.Status,
			AbsentTimes:      candidate.AbsentTimes,
		},
	})
}
