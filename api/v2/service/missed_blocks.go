package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// MissedBlocks returns missed blocks by validator public key.
func (s *Service) MissedBlocks(ctx context.Context, req *pb.MissedBlocksRequest) (*pb.MissedBlocksResponse, error) {
	if !strings.HasPrefix(req.PublicKey, "Mp") {
		return new(pb.MissedBlocksResponse), status.Error(codes.InvalidArgument, "public key don't has prefix 'Mp'")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Lock()
		cState.Validators().LoadValidators()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	val := cState.Validators().GetByPublicKey(types.HexToPubkey(req.PublicKey))
	if val == nil {
		return new(pb.MissedBlocksResponse), status.Error(codes.NotFound, "Validator not found")
	}

	return &pb.MissedBlocksResponse{
		MissedBlocks:      val.AbsentTimes.String(),
		MissedBlocksCount: fmt.Sprintf("%d", val.CountAbsentTimes()),
	}, nil
}
