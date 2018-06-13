package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
	"net/http"
)

func EstimateCoinExchangeReturn(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()
	fromCoin := query.Get("from_coin")
	toCoin := query.Get("to_coin")
	value, _ := big.NewInt(0).SetString(query.Get("value"), 10)

	var fromCoinSymbol types.CoinSymbol
	copy(fromCoinSymbol[:], []byte(fromCoin))

	var toCoinSymbol types.CoinSymbol
	copy(toCoinSymbol[:], []byte(toCoin))

	var result *big.Int

	if fromCoinSymbol == blockchain.BaseCoin {
		coin := blockchain.CurrentState().GetStateCoin(toCoinSymbol).Data()
		result = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, value)
	} else if toCoinSymbol == blockchain.BaseCoin {
		coin := blockchain.CurrentState().GetStateCoin(fromCoinSymbol).Data()
		result = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, value)
	} else {
		coinFrom := blockchain.CurrentState().GetStateCoin(fromCoinSymbol).Data()
		coinTo := blockchain.CurrentState().GetStateCoin(toCoinSymbol).Data()

		val := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, value)
		result = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, val)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result.String(),
	})
}
