package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Validators(_ context.Context, req *pb.ValidatorsRequest) (*pb.ValidatorsResponse, error) {
	height := uint64(req.Height)
	if height == 0 {
		height = s.blockchain.Height()
	}

	h := int64(height)
	tmVals, err := s.client.Validators(&h)
	if err != nil {
		return new(pb.ValidatorsResponse), status.Error(codes.FailedPrecondition, err.Error())
	}

	responseValidators := make([]*pb.ValidatorsResponse_Result, 0, len(tmVals.Validators))
	for _, val := range tmVals.Validators {
		var pk types.Pubkey
		copy(pk[:], val.PubKey.Bytes()[5:])
		responseValidators = append(responseValidators, &pb.ValidatorsResponse_Result{
			PublicKey:   pk.String(),
			VotingPower: fmt.Sprintf("%d", val.VotingPower),
		})
	}
	return &pb.ValidatorsResponse{Validators: responseValidators}, nil
}
