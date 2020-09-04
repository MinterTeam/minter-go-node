package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

func (s *Service) EstimateTxCommission(ctx context.Context, req *pb.EstimateTxCommissionRequest) (*pb.EstimateTxCommissionResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	if !strings.HasPrefix(strings.Title(req.GetTx()), "0x") {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, "invalid transaction")
	}

	decodeString, err := hex.DecodeString(req.GetTx()[2:])
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytesWithoutSig(decodeString)
	if err != nil {
		return new(pb.EstimateTxCommissionResponse), status.Errorf(codes.InvalidArgument, "Cannot decode transaction: %s", err.Error())
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedTx.GasCoin.IsBaseCoin() {
		coin := cState.Coins().GetCoin(decodedTx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateTxCommissionResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"commission_in_base_coin": commissionInBaseCoin.String(),
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
