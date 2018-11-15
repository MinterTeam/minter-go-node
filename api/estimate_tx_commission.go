package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"math/big"
	"net/http"
)

func EstimateTxCommission(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query()
	rawTx := query.Get("tx")
	bytesTx, _ := hexutil.Decode("Mx" + rawTx)

	tx, err := transaction.DecodeFromBytes(bytesTx)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(Response{
			Code: 1,
			Log:  err.Error(),
		})
		return
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := cState.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {

			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(Response{
				Code: 1,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String()),
			})
			return
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Response{
		Code: 0,
		Result: struct {
			Commission string `json:"commission"`
		}{
			Commission: commission.String(),
		},
	})
}
