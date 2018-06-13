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

	vars := mux.Vars(r)
	symbol := vars["symbol"]

	var coinSymbol types.CoinSymbol

	copy(coinSymbol[:], []byte(symbol))

	coin := blockchain.CurrentState().GetStateCoin(coinSymbol).Data()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: CoinInfoResponse{
			Name:           coin.Name,
			Symbol:         coin.Symbol,
			Volume:         coin.Volume.String(),
			Crr:            coin.Crr,
			ReserveCoin:    coin.ReserveCoin,
			ReserveBalance: coin.ReserveBalance.String(),
			Creator:        coin.Creator,
		},
	})
}
