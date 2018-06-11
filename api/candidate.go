package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"minter/core/state"
	"minter/core/types"
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
			CandidateAddress types.Address
			TotalStake       string
			PubKey           string
			Commission       uint
			AccumReward      string
			Stakes           []state.Stake
			CreatedAtBlock   uint
			Status           byte
			AbsentTimes      uint
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
