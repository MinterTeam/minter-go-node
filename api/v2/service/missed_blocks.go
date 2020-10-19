package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

// MissedBlocks returns missed blocks by validator public key.
func (s *Service) MissedBlocks(ctx context.Context, req *pb.MissedBlocksRequest) (*pb.MissedBlocksResponse, error) {
	if !strings.HasPrefix(req.PublicKey, "Mp") {
		return nil, status.Error(codes.InvalidArgument, "public key don't has prefix 'Mp'")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
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
		return nil, status.Error(codes.NotFound, "Validator not found")
	}

	return &pb.MissedBlocksResponse{
		MissedBlocks:      val.AbsentTimes.String(),
		MissedBlocksCount: int64(val.CountAbsentTimes()),
	}, nil
}
