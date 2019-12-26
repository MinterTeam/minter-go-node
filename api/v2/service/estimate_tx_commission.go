package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) EstimateTxCommission(_ context.Context, req *pb.EstimateTxCommissionRequest) (*pb.EstimateTxCommissionResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.EstimateTxCommissionResponse{}, status.Error(codes.NotFound, err.Error())
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytes([]byte(req.Tx))
	if err != nil {
		return &pb.EstimateTxCommissionResponse{}, status.Error(codes.InvalidArgument, "Cannot decode transaction")
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.Coins.GetCoin(decodedTx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return &pb.EstimateTxCommissionResponse{}, s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), encodeError(map[string]string{
				"commission_in_base_coin": coin.Reserve().String(),
				"value_has":               coin.Reserve().String(),
				"value_required":          commissionInBaseCoin.String(),
			}))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	return &pb.EstimateTxCommissionResponse{
		Commission: commission.String(),
	}, nil
}
