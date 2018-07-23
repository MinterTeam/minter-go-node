package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"net/http"
)

type TransactionCountResponse struct {
	Count uint64 `json:"count"`
}

func GetTransactionCount(w http.ResponseWriter, r *http.Request) {

	cState := GetStateForRequest(r)

	vars := mux.Vars(r)
	address := types.HexToAddress(vars["address"])

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: TransactionCountResponse{
			Count: cState.GetNonce(address),
		},
	})
}
