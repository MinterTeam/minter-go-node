package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/golang/protobuf/ptypes/empty"
)

func (s *Service) MinGasPrice(context.Context, *empty.Empty) (*pb.MinGasPriceResponse, error) {
	return &pb.MinGasPriceResponse{
		Jsonrpc: "2.0",
		Id:      "",
		Result:  fmt.Sprintf("%d", s.blockchain.MinGasPrice()),
	}, nil
}

func (s *Service) MaxGas(_ context.Context, req *pb.MaxGasRequest) (*pb.MaxGasResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.MaxGasResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	return &pb.MaxGasResponse{
		Result: fmt.Sprintf("%d", cState.App.GetMaxGas()),
	}, nil
}
