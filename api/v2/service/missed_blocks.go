package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
)

func (s *Service) MissedBlocks(ctx context.Context, req *pb.MissedBlocksRequest) (*pb.MissedBlocksResponse, error) {
	blocks, count, err := s.blockchain.MissedBlocks(req.PublicKey, req.Height)
	if err != nil {
		return new(pb.MissedBlocksResponse), err
	}

	return &pb.MissedBlocksResponse{MissedBlocks: blocks, MissedBlocksCount: fmt.Sprintf("%d", count)}, nil
}
