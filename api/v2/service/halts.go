package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
)

// Halts returns votes
func (s *Service) Halts(_ context.Context, req *pb.HaltsRequest) (*pb.HaltsResponse, error) {
	cState := s.blockchain.CurrentState()

	blocks := cState.Halts().GetHaltBlocks(req.Height)

	if blocks == nil {
		return &pb.HaltsResponse{}, nil
	}

	var votes []string
	for _, vote := range blocks.List {
		votes = append(votes, vote.Pubkey.String())
	}

	return &pb.HaltsResponse{
		PublicKeys: votes,
	}, nil
}
