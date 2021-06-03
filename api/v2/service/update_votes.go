package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UpdateVotes returns votes of new tx commissions.
func (s *Service) UpdateVotes(ctx context.Context, req *pb.UpdateVotesRequest) (*pb.UpdateVotesResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	votes := cState.Updates().GetVotes(req.TargetVersion)
	if len(votes) == 0 {
		return &pb.UpdateVotesResponse{}, nil
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	resp := make([]*pb.UpdateVotesResponse_Vote, 0, len(votes))
	for _, vote := range votes {
		pubKeys := make([]string, 0, len(vote.Votes))
		for _, pubkey := range vote.Votes {
			pubKeys = append(pubKeys, pubkey.String())
		}
		resp = append(resp, &pb.UpdateVotesResponse_Vote{
			Version:    vote.Version,
			PublicKeys: pubKeys,
		})
	}

	return &pb.UpdateVotesResponse{
		Votes: resp,
	}, nil
}
