package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
)

// PriceVotes returns votes of new tx commissions.
func (s *Service) PriceVotes(ctx context.Context, req *pb.PriceVotesRequest) (*pb.PriceVotesResponse, error) {
	cState := s.blockchain.CurrentState()

	votes := cState.Commission().GetVotes(req.Height)

	if len(votes) == 0 {
		return &pb.PriceVotesResponse{}, nil
	}

	resp := make([]*pb.PriceVotesResponse_Vote, 0, len(votes))
	for _, vote := range votes {
		pubKeys := make([]string, 0, len(vote.Votes))
		for _, pubkey := range vote.Votes {
			pubKeys = append(pubKeys, pubkey.String())
		}
		price := commission.Decode(vote.Price)
		resp = append(resp, &pb.PriceVotesResponse_Vote{
			Price:      priceCommissionResponse(price, cState.Coins().GetCoin(price.Coin)),
			PublicKeys: pubKeys,
		})
	}

	return &pb.PriceVotesResponse{
		PriceVotes: resp,
	}, nil
}
