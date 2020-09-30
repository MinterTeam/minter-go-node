package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MinGasPrice returns current min gas price.
func (s *Service) MinGasPrice(context.Context, *empty.Empty) (*pb.MinGasPriceResponse, error) {
	return &pb.MinGasPriceResponse{
		MinGasPrice: uint64(s.blockchain.MinGasPrice()),
	}, nil
}

// MaxGas returns current max gas.
func (s *Service) MaxGasPrice(ctx context.Context, req *pb.MaxGasPriceRequest) (*pb.MaxGasPriceResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &pb.MaxGasPriceResponse{
		MaxGasPrice: cState.App().GetMaxGas(),
	}, nil
}
