package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validators returns list of active validators.
func (s *Service) Validators(ctx context.Context, req *pb.ValidatorsRequest) (*pb.ValidatorsResponse, error) {
	height := int64(req.Height)
	if height == 0 {
		height = int64(s.blockchain.Height())
	}
	var page = 1
	var perPage = 100
	tmVals, err := s.client.Validators(ctx, &height, &page, &perPage)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	responseValidators := make([]*pb.ValidatorsResponse_Result, 0, len(tmVals.Validators))
	for _, val := range tmVals.Validators {
		var pk types.Pubkey
		copy(pk[:], val.PubKey.Bytes()[:])
		responseValidators = append(responseValidators, &pb.ValidatorsResponse_Result{
			PublicKey:   pk.String(),
			VotingPower: uint64(val.VotingPower),
		})
	}
	return &pb.ValidatorsResponse{Validators: responseValidators}, nil
}
