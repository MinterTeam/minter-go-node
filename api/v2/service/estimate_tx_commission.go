package service

import (
	"context"
	"encoding/hex"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) EstimateTxCommission(_ context.Context, req *pb.EstimateTxCommissionRequest) (*pb.EstimateTxCommissionResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	if len(req.Tx) < 3 {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "invalid tx")
	}

	decodeString, err := hex.DecodeString(req.Tx[2:])
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytesWithoutSig(decodeString)
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "Cannot decode transaction")
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.Coins.GetCoin(decodedTx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateTxCommissionResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
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
