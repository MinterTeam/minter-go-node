package api

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"minter/core/types"
	"math/big"
)

type CoinInfoResponse struct {
	Name           string           `json:"name"`
	Symbol         types.CoinSymbol `json:"symbol"`
	Volume         *big.Int         `json:"volume"`
	Crr            uint             `json:"crr"`
	ReserveCoin    types.CoinSymbol `json:"reserve_coin"`
	ReserveBalance *big.Int         `json:"reserve_balance"`
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
			Volume:         coin.Volume,
			Crr:            coin.Crr,
			ReserveCoin:    coin.ReserveCoin,
			ReserveBalance: coin.ReserveBalance,
			Creator:        coin.Creator,
		},
	})
}
