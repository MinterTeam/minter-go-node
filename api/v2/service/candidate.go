package service

import (
	"context"
	"encoding/hex"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) Candidate(_ context.Context, req *pb.CandidateRequest) (*pb.CandidateResponse, error) {
	if len(req.PublicKey) < 3 {
		return new(pb.CandidateResponse), status.Error(codes.InvalidArgument, "invalid public_key")
	}
	decodeString, err := hex.DecodeString(req.PublicKey[2:])
	if err != nil {
		return new(pb.CandidateResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.CandidateResponse), status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		cState.Candidates.LoadStakes()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidate := cState.Candidates.GetCandidate(types.BytesToPubkey(decodeString))
	if candidate == nil {
		return new(pb.CandidateResponse), status.Error(codes.NotFound, "Candidate not found")
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
		candidate.Stakes = make([]*pb.CandidateResponse_Stake, 0, len(stakes))
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
