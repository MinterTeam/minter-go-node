package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

func (s *Service) EstimateTxCommission(_ context.Context, req *pb.EstimateTxCommissionRequest) (*pb.EstimateTxCommissionResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.EstimateTxCommissionResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytes([]byte(req.Tx))
	if err != nil {
		return &pb.EstimateTxCommissionResponse{
			Error: &pb.Error{
				Code:    "400",
				Message: "Cannot decode transaction",
				Data:    err.Error(),
			},
		}, nil
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.Coins.GetCoin(decodedTx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return &pb.EstimateTxCommissionResponse{
				Error: &pb.Error{
					Code: "400",
					Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
						coin.Reserve().String(), commissionInBaseCoin.String()),
				},
			}, nil
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	return &pb.EstimateTxCommissionResponse{
		Result: &pb.EstimateTxCommissionResponse_Result{
			Commission: commission.String(),
		},
	}, nil
}
