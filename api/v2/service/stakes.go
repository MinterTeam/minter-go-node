package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (s *Service) Stakes(ctx context.Context, req *pb.StakesRequest) (*pb.StakesResponse, error) {
	if !strings.HasPrefix(req.PublicKey, "Mp") {
		return new(pb.StakesResponse), status.Error(codes.InvalidArgument, "public key don't has prefix 'Mp'")
	}

	pubKey := types.HexToPubkey(req.PublicKey[2:])

	var address types.Address
	if req.Address != "" {
		if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
			return new(pb.StakesResponse), status.Error(codes.InvalidArgument, "invalid address")
		}

		decodeAddr, err := hex.DecodeString(req.Address[2:])
		if err != nil {
			return new(pb.StakesResponse), status.Error(codes.InvalidArgument, "invalid address")
		}

		address = types.BytesToAddress(decodeAddr)
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, err
	}

	if req.Height != 0 {
		cState.Lock()
		cState.Candidates().LoadCandidates()
		cState.Candidates().LoadStakes()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	var allCandidates []*candidates.Candidate
	if req.PublicKey == "" {
		allCandidates = cState.Candidates().GetCandidates()
	} else {
		allCandidates = []*candidates.Candidate{cState.Candidates().GetCandidate(pubKey)}
	}

	var result []*pb.StakesResponse_Stake
	for _, candidate := range allCandidates {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return new(pb.StakesResponse), timeoutStatus.Err()
		}

		var multi bool
		var allPubkeyStakes bool

		if req.Coin != "" && req.Address != "" {
			multi = true
		}
		if req.Coin == "" && req.Address == "" {
			allPubkeyStakes = true
		}

		stakes := cState.Candidates().GetStakes(pubKey)
		for _, stake := range stakes {
			if !((multi && stake.Coin.String() == req.Coin && stake.Owner == address) || (!multi && (stake.Coin.String() == req.Coin || stake.Owner == address || allPubkeyStakes))) {
				continue
			}
			result = append(result, &pb.StakesResponse_Stake{
				Address:  stake.Owner.String(),
				PubKey:   candidate.PubKey.String(),
				Coin:     stake.Coin.String(),
				Value:    stake.Value.String(),
				BipValue: stake.BipValue.String(),
			})
		}
	}

	return &pb.StakesResponse{
		Stakes: result,
	}, nil
}
