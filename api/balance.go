package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"net/http"
)

type BalanceResponse map[string]string

type BalanceRequest struct {
	Address types.Address    `json:"address"`
	Coin    types.CoinSymbol `json:"coin"`
}

func GetBalance(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	address := types.HexToAddress(vars["address"])

	balance := BalanceResponse{}
	balances := blockchain.CurrentState().GetBalances(address)

	for k, v := range balances.Data {
		balance[k.String()] = v.String()
	}

	if _, exists := balance[types.GetBaseCoin().String()]; !exists {
		balance[types.GetBaseCoin().String()] = "0"
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: balance,
	})
}
