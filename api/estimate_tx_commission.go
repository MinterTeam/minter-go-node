package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
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
		return nil, rpctypes.RPCError{Code: 400, Message: "Cannot decode transaction", Data: err.Error()}
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.Coins.GetCoin(decodedTx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	return &TxCommissionResponse{
		Commission: commission,
	}, nil
}
