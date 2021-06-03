package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CommissionVotes returns votes of new tx commissions.
func (s *Service) CommissionVotes(ctx context.Context, req *pb.CommissionVotesRequest) (*pb.CommissionVotesResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	votes := cState.Commission().GetVotes(req.TargetVersion)
	if len(votes) == 0 {
		return &pb.CommissionVotesResponse{}, nil
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	resp := make([]*pb.CommissionVotesResponse_Vote, 0, len(votes))
	for _, vote := range votes {
		pubKeys := make([]string, 0, len(vote.Votes))
		for _, pubkey := range vote.Votes {
			pubKeys = append(pubKeys, pubkey.String())
		}
		price := commission.Decode(vote.Price)
		resp = append(resp, &pb.CommissionVotesResponse_Vote{
			Price:      priceCommissionResponse(price, cState.Coins().GetCoin(price.Coin)),
			PublicKeys: pubKeys,
		})
	}

	return &pb.CommissionVotesResponse{
		Votes: resp,
	}, nil
}
