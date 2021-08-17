package service

import (
	"context"

	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Blocks ...
func (s *Service) Blocks(ctx context.Context, req *pb.BlocksRequest) (*pb.BlocksResponse, error) {
	const limit = 100

	var blocksResponse []*pb.BlockResponse
	for i := req.FromHeight; i <= req.ToHeight && i <= req.FromHeight+limit; i++ {
		block, err := s.Block(ctx, &pb.BlockRequest{
			Height:    i,
			Fields:    req.Fields,
			FailedTxs: req.FailedTxs,
			Events:    req.Events,
		})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				break
			}
			return nil, err
		}
		blocksResponse = append(blocksResponse, block)
	}

	return &pb.BlocksResponse{Blocks: blocksResponse}, nil
}
