package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) MinGasPrice(context.Context, *empty.Empty) (*pb.MinGasPriceResponse, error) {
	return &pb.MinGasPriceResponse{
		Result: fmt.Sprintf("%d", s.blockchain.MinGasPrice()),
	}, nil
}

func (s *Service) MaxGas(_ context.Context, req *pb.MaxGasRequest) (*pb.MaxGasResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.MaxGasResponse), status.Error(codes.NotFound, err.Error())
	}

	return &pb.MaxGasResponse{
		Result: fmt.Sprintf("%d", cState.App.GetMaxGas()),
	}, nil
}
