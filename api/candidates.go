package api

import (
	"encoding/json"
	"net/http"
)

func GetCandidates(w http.ResponseWriter, r *http.Request) {

	cState := GetStateForRequest(r)

	candidates := cState.GetStateCandidates().GetData()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var result []Candidate

	for _, candidate := range candidates {
		result = append(result, makeResponseCandidate(candidate))
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result,
	})
}
