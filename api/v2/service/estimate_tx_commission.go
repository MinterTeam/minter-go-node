package service

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/MinterTeam/minter-go-node/core/transaction"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EstimateTxCommission return estimate of transaction.
func (s *Service) EstimateTxCommission(ctx context.Context, req *pb.EstimateTxCommissionRequest) (*pb.EstimateTxCommissionResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if !strings.HasPrefix(strings.Title(req.GetTx()), "0x") {
		return nil, status.Error(codes.InvalidArgument, "invalid transaction")
	}

	decodeString, err := hex.DecodeString(req.GetTx()[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	decodedTx, err := transaction.TxDecoder.DecodeFromBytesWithoutSig(decodeString)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Cannot decode transaction: %s", err.Error())
	}

	commissionInBaseCoin := decodedTx.CommissionInBaseCoin()
	commission, _, errResp := transaction.CalculateCommission(cState, cState.Coins().GetCoin(decodedTx.GasCoin), commissionInBaseCoin)
	if errResp != nil {
		return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
	}

	return &pb.EstimateTxCommissionResponse{
		Commission: commission.String(),
	}, nil
}
