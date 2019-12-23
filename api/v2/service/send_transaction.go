package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
)

func (s *Service) SendTransaction(_ context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	result, err := s.client.BroadcastTxSync([]byte(req.Tx))
	if err != nil {
		return &pb.SendTransactionResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	if result.Code != 0 {
		return &pb.SendTransactionResponse{
			Error: &pb.Error{
				Code: fmt.Sprintf("%d", result.Code),
				Log:  result.Log,
			},
		}, nil
	}

	return &pb.SendTransactionResponse{
		Code: fmt.Sprintf("%d", result.Code),
		Data: result.Data.String(),
		Log:  result.Log,
		Hash: result.Hash.String(),
	}, nil
}
