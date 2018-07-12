package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
	"net/http"
)

func EstimateCoinExchangeReturn(w http.ResponseWriter, r *http.Request) {

	cState := GetStateForRequest(r)

	query := r.URL.Query()
	fromCoin := query.Get("from_coin")
	toCoin := query.Get("to_coin")
	value, _ := big.NewInt(0).SetString(query.Get("value"), 10)

	var fromCoinSymbol types.CoinSymbol
	copy(fromCoinSymbol[:], []byte(fromCoin))

	var toCoinSymbol types.CoinSymbol
	copy(toCoinSymbol[:], []byte(toCoin))

	var result *big.Int

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if fromCoinSymbol == blockchain.BaseCoin {
		coin := cState.GetStateCoin(toCoinSymbol)
		if coin == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(Response{
				Code:   404,
				Result: nil,
				Log:    "Coin not found",
			})
			return
		}
		result = formula.CalculatePurchaseReturn(coin.Data().Volume, coin.Data().ReserveBalance, coin.Data().Crr, value)
	} else if toCoinSymbol == blockchain.BaseCoin {
		coin := cState.GetStateCoin(fromCoinSymbol)
		if coin == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(Response{
				Code:   404,
				Result: nil,
				Log:    "Coin not found",
			})
			return
		}
		result = formula.CalculateSaleReturn(coin.Data().Volume, coin.Data().ReserveBalance, coin.Data().Crr, value)
	} else {
		coinFrom := cState.GetStateCoin(fromCoinSymbol)
		coinTo := cState.GetStateCoin(toCoinSymbol)

		if coinFrom == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(Response{
				Code:   404,
				Result: nil,
				Log:    "Coin not found",
			})
			return
		}

		if coinTo == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(Response{
				Code:   404,
				Result: nil,
				Log:    "Coin not found",
			})
			return
		}

		val := formula.CalculateSaleReturn(coinFrom.Data().Volume, coinFrom.Data().ReserveBalance, coinFrom.Data().Crr, value)
		result = formula.CalculatePurchaseReturn(coinTo.Data().Volume, coinTo.Data().ReserveBalance, coinTo.Data().Crr, val)
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result.String(),
	})
}
