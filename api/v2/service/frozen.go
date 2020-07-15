package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (s *Service) Frozen(ctx context.Context, req *pb.FrozenRequest) (*pb.FrozenResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Address), "Mx") {
		return new(pb.FrozenResponse), status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Address[2:])
	if err != nil {
		return new(pb.FrozenResponse), status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)

	cState := s.blockchain.CurrentState()
	cState.RLock()
	defer cState.RUnlock()

	var frozen []*pb.FrozenResponse_Frozen

	appState := new(types.AppState)
	cState.FrozenFunds().Export(appState, s.blockchain.Height())

	var emptyAddress types.Address

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.FrozenResponse), timeoutStatus.Err()
	}

	if req.Coin == "" && address == emptyAddress {
		for _, fund := range appState.FrozenFunds {
			frozen = append(frozen, &pb.FrozenResponse_Frozen{
				Height:       fmt.Sprintf("%d", fund.Height),
				Address:      fund.Address.String(),
				CandidateKey: fund.CandidateKey.String(),
				Coin:         fund.Coin.String(),
				Value:        fund.Value,
			})
		}
		return &pb.FrozenResponse{Frozen: frozen}, nil
	}

	if req.Coin != "" && address != emptyAddress {
		for _, fund := range appState.FrozenFunds {
			if fund.Coin.String() != req.Coin || fund.Address != address {
				continue
			}
			frozen = append(frozen, &pb.FrozenResponse_Frozen{
				Height:       fmt.Sprintf("%d", fund.Height),
				Address:      fund.Address.String(),
				CandidateKey: fund.CandidateKey.String(),
				Coin:         fund.Coin.String(),
				Value:        fund.Value,
			})
		}
		return &pb.FrozenResponse{Frozen: frozen}, nil
	}

	for _, fund := range appState.FrozenFunds {
		if fund.Coin.String() != req.Coin && fund.Address != address {
			continue
		}
		frozen = append(frozen, &pb.FrozenResponse_Frozen{
			Height:       fmt.Sprintf("%d", fund.Height),
			Address:      fund.Address.String(),
			CandidateKey: fund.CandidateKey.String(),
			Coin:         fund.Coin.String(),
			Value:        fund.Value,
		})
	}
	return &pb.FrozenResponse{Frozen: frozen}, nil
}
