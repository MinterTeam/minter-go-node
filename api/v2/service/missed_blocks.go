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
	blocks, count, err := missedBlocks(s, req.Height, req.PublicKey)
	if err != nil {
		return new(pb.MissedBlocksResponse), err
	}

	return &pb.MissedBlocksResponse{MissedBlocks: blocks, MissedBlocksCount: fmt.Sprintf("%d", count)}, nil
}

func missedBlocks(s *Service, height uint64, pubKey string) (missedBlocks string, missedBlocksCount int, err error) {
	if !strings.HasPrefix(pubKey, "Mp") {
		return "", 0, status.Error(codes.InvalidArgument, "public key don't has prefix 'Mp'")
	}

	cState, err := s.blockchain.GetStateForHeight(height)
	if err != nil {
		return "", 0, status.Error(codes.NotFound, err.Error())
	}

	if height != 0 {
		cState.Lock()
		cState.Validators().LoadValidators()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	val := cState.Validators().GetByPublicKey(types.HexToPubkey(pubKey))
	if val == nil {
		return "", 0, status.Error(codes.NotFound, "Validator not found")
	}

	return val.AbsentTimes.String(), val.CountAbsentTimes(), nil
}
