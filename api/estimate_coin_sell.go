package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
	"net/http"
)

type EstimateCoinSellResponse struct {
	WillGet    string `json:"will_get"`
	Commission string `json:"commission"`
}

func EstimateCoinSell(w http.ResponseWriter, r *http.Request) {

	cState := GetStateForRequest(r)

	query := r.URL.Query()
	coinToSell := query.Get("coin_to_sell")
	coinToBuy := query.Get("coin_to_buy")
	valueToSell, _ := big.NewInt(0).SetString(query.Get("value_to_sell"), 10)

	var coinToSellSymbol types.CoinSymbol
	copy(coinToSellSymbol[:], []byte(coinToSell))

	var coinToBuySymbol types.CoinSymbol
	copy(coinToBuySymbol[:], []byte(coinToBuy))

	var result *big.Int

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if coinToSell == coinToBuy {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin"),
		})
		return
	}

	if !cState.CoinExists(coinToSellSymbol) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("Coin to sell not exists"),
		})
		return
	}

	if !cState.CoinExists(coinToBuySymbol) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("Coin to buy not exists"),
		})
		return
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if coinToSellSymbol != types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToSellSymbol)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{
				Code: 1,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String()),
			})
			return
		}

		if coin.Volume().Cmp(valueToSell) < 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(Response{
				Code: 1,
				Log:  fmt.Sprintf("Coin volume is not sufficient for transaction. Has: %s, required %s", coin.Volume().String(), valueToSell.String()),
			})
			return
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if coinToSellSymbol == types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToBuySymbol).Data()
		result = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, valueToSell)
	} else if coinToBuySymbol == types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToSellSymbol).Data()
		result = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, valueToSell)
	} else {
		coinFrom := cState.GetStateCoin(coinToSellSymbol).Data()
		coinTo := cState.GetStateCoin(coinToBuySymbol).Data()
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, valueToSell)
		result = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: EstimateCoinSellResponse{
			WillGet:    result.String(),
			Commission: commission.String(),
		},
	})
}
