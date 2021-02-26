package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/candidates"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"math/big"
	"strings"
)

// Candidate returns candidate’s info by provided public_key. It will respond with 404 code if candidate is not found.
func (s *Service) Candidate(ctx context.Context, req *pb.CandidateRequest) (*pb.CandidateResponse, error) {
	if !strings.HasPrefix(req.PublicKey, "Mp") {
		return nil, status.Error(codes.InvalidArgument, "invalid public_key")
	}

	decodeString, err := hex.DecodeString(req.PublicKey[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pubkey := types.BytesToPubkey(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if req.Height != 0 {
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakesOfCandidate(pubkey)
	}

	candidate := cState.Candidates().GetCandidate(pubkey)
	if candidate == nil {
		return nil, status.Error(codes.NotFound, "Candidate not found")
	}

	result := makeResponseCandidate(cState, candidate, true, req.NotShowStakes)
	if cState.Validators().GetByPublicKey(candidate.PubKey) != nil {
		result.Validator = true
	}

	return result, nil
}

func makeResponseCandidate(state *state.CheckState, c *candidates.Candidate, includeStakes, NotShowStakes bool) *pb.CandidateResponse {
	candidate := &pb.CandidateResponse{
		RewardAddress:  c.RewardAddress.String(),
		OwnerAddress:   c.OwnerAddress.String(),
		ControlAddress: c.ControlAddress.String(),
		TotalStake:     state.Candidates().GetTotalStake(c.PubKey).String(),
		PublicKey:      c.PubKey.String(),
		Commission:     uint64(c.Commission),
		Status:         uint64(c.Status),
	}

	if includeStakes {
		state.Candidates().LoadStakesOfCandidate(c.PubKey)
		stakes := state.Candidates().GetStakes(c.PubKey)
		addresses := map[types.Address]struct{}{}
		minStake := big.NewInt(0)
		usedSlots := len(stakes)
		if !NotShowStakes {
			candidate.Stakes = make([]*pb.CandidateResponse_Stake, 0, usedSlots)
		}
		for i, stake := range stakes {
			if !NotShowStakes {
				candidate.Stakes = append(candidate.Stakes, &pb.CandidateResponse_Stake{
					Owner: stake.Owner.String(),
					Coin: &pb.Coin{
						Id:     uint64(stake.Coin),
						Symbol: state.Coins().GetCoin(stake.Coin).GetFullSymbol(),
					},
					Value:    stake.Value.String(),
					BipValue: stake.BipValue.String(),
				})
			}
			addresses[stake.Owner] = struct{}{}
			if usedSlots >= candidates.MaxDelegatorsPerCandidate {
				if i != 0 && minStake.Cmp(stake.BipValue) != 1 {
					continue
				}
				minStake = stake.BipValue
			}
		}
		candidate.UsedSlots = wrapperspb.UInt64(uint64(usedSlots))
		candidate.UniqUsers = wrapperspb.UInt64(uint64(len(addresses)))
		candidate.MinStake = wrapperspb.String(minStake.String())
	}

	return candidate
}
