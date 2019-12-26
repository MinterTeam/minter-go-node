package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) MissedBlocks(_ context.Context, req *pb.MissedBlocksRequest) (*pb.MissedBlocksResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, err.Error())
	}

	vals := cState.Validators.GetValidators()
	if vals == nil {
		return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, "Validators not found")
	}

	for _, val := range vals {
		if string(val.PubKey[:]) == req.PublicKey {
			return &pb.MissedBlocksResponse{
				MissedBlocks:      val.AbsentTimes.String(),
				MissedBlocksCount: fmt.Sprintf("%d", val.CountAbsentTimes()),
			}, nil
		}
	}

	return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, "Validator not found")

}
