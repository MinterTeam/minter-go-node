package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Status returns current min gas price.
func (s *Service) VersionNetwork(ctx context.Context, req *pb.VersionNetworkRequest) (*pb.VersionNetworkResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &pb.VersionNetworkResponse{
		Version: cState.App().GetVersion(),
	}, nil
}
