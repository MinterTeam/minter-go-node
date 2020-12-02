package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (s *Service) Pair(_ context.Context, req *pb.PairRequest) (*pb.PairResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}
	state := s.blockchain.CurrentState()
	totalSupply, reserve0, reserve1 := state.Swap().PairInfo(types.CoinID(req.Coin1))
	if totalSupply == nil {
		return nil, status.Error(codes.NotFound, "pair not found")
	}
	return &pb.PairResponse{
		Reserve0:    reserve0.String(),
		Reserve1:    reserve1.String(),
		TotalSupply: totalSupply.String(),
	}, nil
}

func (s *Service) PairFromProvider(_ context.Context, req *pb.PairFromProviderRequest) (*pb.PairFromProviderResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Provider), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	decodeString, err := hex.DecodeString(req.Provider[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)
	state := s.blockchain.CurrentState()
	balance, amount0, amount1 := state.Swap().PairFromProvider(address, types.CoinID(req.Coin1))
	if balance == nil {
		return nil, status.Error(codes.NotFound, "pair from provider not found")
	}
	return &pb.PairFromProviderResponse{
		Amount0: amount0.String(),
		Amount1: amount1.String(),
		Balance: balance.String(),
	}, nil
}
