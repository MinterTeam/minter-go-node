package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/pkg/errors"
	"math/big"
)

type TxCommissionResponse struct {
	Commission *big.Int `json:"commission"`
}

func EstimateTxCommission(tx []byte, height int) (*TxCommissionResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytes(tx)
	if err != nil {
		return nil, err
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.GetStateCoin(decodedTx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return nil, errors.New(fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String()))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	return &TxCommissionResponse{
		Commission: commission,
	}, nil
}
