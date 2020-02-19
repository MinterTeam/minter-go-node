package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) MissedBlocks(_ context.Context, req *pb.MissedBlocksRequest) (*pb.MissedBlocksResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.Lock()
	defer cState.Unlock()

	if req.Height != 0 {
		cState.Validators.LoadValidators()
	}

	vals := cState.Validators.GetValidators()
	if vals == nil {
		return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, "Validators not found")
	}

	for _, val := range vals {
		if len(req.PublicKey) < 3 {
			return new(pb.MissedBlocksResponse), status.Error(codes.InvalidArgument, "invalid public_key")
		}
		decodeString, err := hex.DecodeString(req.PublicKey[2:])
		if err != nil {
			return new(pb.MissedBlocksResponse), status.Error(codes.InvalidArgument, err.Error())
		}
		if bytes.Compare(val.PubKey[:], decodeString) == 0 {
			return &pb.MissedBlocksResponse{
				MissedBlocks:      val.AbsentTimes.String(),
				MissedBlocksCount: fmt.Sprintf("%d", val.CountAbsentTimes()),
			}, nil
		}
	}

	return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, "Validator not found")

}
