package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"net/http"
	"strings"
)

func GetCandidate(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	pubkey := types.Hex2Bytes(strings.TrimLeft(vars["pubkey"], "Mp"))

	cState := GetStateForRequest(r)

	candidate := cState.GetStateCandidate(pubkey)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if candidate == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{
			Code: 404,
			Log:  "Candidate not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: struct {
			Candidate Candidate `json:"candidate"`
		}{
			Candidate: makeResponseCandidate(*candidate),
		},
	})
}
