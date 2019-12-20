package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
)

func (s *Service) MissedBlocks(_ context.Context, req *pb.MissedBlocksRequest) (*pb.MissedBlocksResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.MissedBlocksResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	vals := cState.Validators.GetValidators()
	if vals == nil {
		return &pb.MissedBlocksResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Validators not found",
			},
		}, nil
	}

	for _, val := range vals {
		if string(val.PubKey[:]) == req.PublicKey {
			return &pb.MissedBlocksResponse{
				Result: &pb.MissedBlocksResponse_Result{
					MissedBlocks:      val.AbsentTimes.String(),
					MissedBlocksCount: fmt.Sprintf("%d", val.CountAbsentTimes()),
				},
			}, nil
		}
	}

	return &pb.MissedBlocksResponse{
		Error: &pb.Error{
			Code:    "404",
			Message: "Validator not found",
		},
	}, nil

}
