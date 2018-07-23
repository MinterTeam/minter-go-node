package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"net/http"
)

type BalanceResponse struct {
	Balance map[string]string `json:"balance"`
}

type BalanceRequest struct {
	Address types.Address    `json:"address"`
	Coin    types.CoinSymbol `json:"coin"`
}

func GetBalance(w http.ResponseWriter, r *http.Request) {

	cState := GetStateForRequest(r)

	vars := mux.Vars(r)
	address := types.HexToAddress(vars["address"])

	balanceResponse := BalanceResponse{
		Balance: make(map[string]string),
	}
	balances := cState.GetBalances(address)

	for k, v := range balances.Data {
		balanceResponse.Balance[k.String()] = v.String()
	}

	if _, exists := balanceResponse.Balance[types.GetBaseCoin().String()]; !exists {
		balanceResponse.Balance[types.GetBaseCoin().String()] = "0"
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: balanceResponse,
	})
}
