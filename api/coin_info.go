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
	ReserveCoin    types.CoinSymbol `json:"reserve_coin"`
	ReserveBalance string           `json:"reserve_balance"`
	Creator        types.Address    `json:"creator"`
}

func GetCoinInfo(w http.ResponseWriter, r *http.Request) {

	cState := GetStateForRequest(r)

	vars := mux.Vars(r)
	symbol := vars["symbol"]

	var coinSymbol types.CoinSymbol

	copy(coinSymbol[:], []byte(symbol))

	coin := cState.GetStateCoin(coinSymbol)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if coin == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{
			Code:   404,
			Result: nil,
			Log:    "Coin not found",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: CoinInfoResponse{
			Name:           coin.Data().Name,
			Symbol:         coin.Data().Symbol,
			Volume:         coin.Data().Volume.String(),
			Crr:            coin.Data().Crr,
			ReserveCoin:    coin.Data().ReserveCoin,
			ReserveBalance: coin.Data().ReserveBalance.String(),
			Creator:        coin.Data().Creator,
		},
	})
}
