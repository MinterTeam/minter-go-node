package api

import (
	"encoding/json"
	"math/big"
	"minter/core/types"
	"net/http"
	"github.com/gorilla/mux"
)

type BalanceResponse map[string]*big.Int

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
		balance[k.String()] = v
	}

	if _, exists := balance[types.GetBaseCoin().String()]; !exists {
		balance[types.GetBaseCoin().String()] = types.Big0
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: balance,
	})
}
