package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
)

func (s *Service) Candidate(_ context.Context, req *pb.CandidateRequest) (*pb.CandidateResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.CandidateResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	candidate := cState.Candidates.GetCandidate(types.BytesToPubkey([]byte(req.PublicKey)))
	if candidate == nil {
		return &pb.CandidateResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Candidate not found",
			},
		}, nil
	}

	result := makeResponseCandidate(cState, *candidate, true)
	return result, nil
}

func makeResponseCandidate(state *state.State, c candidates.Candidate, includeStakes bool) *pb.CandidateResponse {
	candidate := &pb.CandidateResponse{
		RewardAddress: c.RewardAddress.String(),
		TotalStake:    state.Candidates.GetTotalStake(c.PubKey).String(),
		PublicKey:     c.PubKey.String(),
		Commission:    fmt.Sprintf("%d", c.Commission),
		Status:        fmt.Sprintf("%d", c.Status),
	}

	if includeStakes {
		stakes := state.Candidates.GetStakes(c.PubKey)
		candidate.Stakes = make([]*pb.CandidateResponse_Stake, len(stakes))
		for _, stake := range stakes {
			candidate.Stakes = append(candidate.Stakes, &pb.CandidateResponse_Stake{
				Owner:    stake.Owner.String(),
				Coin:     stake.Coin.String(),
				Value:    stake.Value.String(),
				BipValue: stake.BipValue.String(),
			})
		}
	}

	return candidate
}
