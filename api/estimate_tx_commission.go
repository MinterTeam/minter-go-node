package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/pkg/errors"
	"math/big"
)

func EstimateTxCommission(rawTx []byte, height int) (*big.Int, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	tx, err := transaction.DecodeFromBytes(rawTx)
	if err != nil {
		return nil, err
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := cState.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return nil, errors.New(fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String()))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	return commission, nil
}
