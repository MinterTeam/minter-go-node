package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/gorilla/mux"
	"net/http"
)

type CoinInfoResponse struct {
	Name           string           `json:"name"`
	Symbol         types.CoinSymbol `json:"symbol"`
	Volume         string           `json:"volume"`
	Crr            uint             `json:"crr"`
	ReserveBalance string           `json:"reserve_balance"`
}

func GetCoinInfo(w http.ResponseWriter, r *http.Request) {
	cState, err := GetStateForRequest(r)

	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(Response{
			Code: 404,
			Log:  "State for given height not found",
		})
		return
	}

	vars := mux.Vars(r)
	symbol := vars["symbol"]

	var coinSymbol types.CoinSymbol

	copy(coinSymbol[:], []byte(symbol))

	coin := cState.GetStateCoin(coinSymbol)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if coin == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(Response{
			Code:   404,
			Result: nil,
			Log:    "Coin not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: CoinInfoResponse{
			Name:           coin.Data().Name,
			Symbol:         coin.Data().Symbol,
			Volume:         coin.Data().Volume.String(),
			Crr:            coin.Data().Crr,
			ReserveBalance: coin.Data().ReserveBalance.String(),
		},
	})
}
