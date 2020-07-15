package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) Candidate(ctx context.Context, req *pb.CandidateRequest) (*pb.CandidateResponse, error) {
	if len(req.PublicKey) < 3 {
		return new(pb.CandidateResponse), status.Error(codes.InvalidArgument, "invalid public_key")
	}
	decodeString, err := hex.DecodeString(req.PublicKey[2:])
	if err != nil {
		return new(pb.CandidateResponse), status.Error(codes.InvalidArgument, err.Error())
	}

	pubkey := types.BytesToPubkey(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.CandidateResponse), status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakesOfCandidate(pubkey)
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidate := cState.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		return new(pb.CandidateResponse), status.Error(codes.NotFound, "Candidate not found")
	}

	result := makeResponseCandidate(cState, *candidate, true)
	return result, nil
}

func makeResponseCandidate(state *state.CheckState, c candidates.Candidate, includeStakes bool) *pb.CandidateResponse {
	candidate := &pb.CandidateResponse{
		RewardAddress: c.RewardAddress.String(),
		TotalStake:    state.Candidates().GetTotalStake(c.PubKey).String(),
		PublicKey:     c.PubKey.String(),
		Commission:    fmt.Sprintf("%d", c.Commission),
		Status:        fmt.Sprintf("%d", c.Status),
	}

	if includeStakes {
		addresses := map[types.Address]struct{}{}
		minStake := big.NewInt(0)
		stakes := state.Candidates().GetStakes(c.PubKey)
		usedSlots := len(stakes)
		candidate.UsedSlots = fmt.Sprintf("%d", usedSlots)
		candidate.Stakes = make([]*pb.CandidateResponse_Stake, 0, usedSlots)
		for i, stake := range stakes {
			candidate.Stakes = append(candidate.Stakes, &pb.CandidateResponse_Stake{
				Owner:    stake.Owner.String(),
				Coin:     stake.Coin.String(),
				Value:    stake.Value.String(),
				BipValue: stake.BipValue.String(),
			})
			addresses[stake.Owner] = struct{}{}
			if usedSlots >= candidates.MaxDelegatorsPerCandidate {
				if i != 0 && minStake.Cmp(stake.BipValue) != 1 {
					continue
				}
				minStake = stake.BipValue
			}
		}
		candidate.UniqUsers = fmt.Sprintf("%d", len(addresses))
		candidate.MinStake = minStake.String()
	}

	return candidate
}
