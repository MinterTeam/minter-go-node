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

func (s *Service) Pair(_ context.Context, req *pb.SwapPoolRequest) (*pb.SwapPoolResponse, error) {
	if req.Token == uint64(types.GetSwapHubCoinID()) {
		return nil, status.Error(codes.NotFound, "swap pool not found")
	}
	state := s.blockchain.CurrentState()
	totalSupply, reserve0, reserve1 := state.Swap().PairInfo(types.CoinID(req.Token))
	if totalSupply == nil {
		return nil, status.Error(codes.NotFound, "swap pool not found")
	}
	return &pb.SwapPoolResponse{
		ReserveHub:   reserve0.String(),
		ReserveToken: reserve1.String(),
		TotalSupply:  totalSupply.String(),
	}, nil
}

func (s *Service) PairFromProvider(_ context.Context, req *pb.SwapPoolFromProviderRequest) (*pb.SwapPoolFromProviderResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Provider), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	if req.Token == uint64(types.GetSwapHubCoinID()) {
		return nil, status.Error(codes.NotFound, "swap pool not found")
	}
	decodeString, err := hex.DecodeString(req.Provider[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)
	state := s.blockchain.CurrentState()
	balance, amount0, amount1 := state.Swap().PairFromProvider(address, types.CoinID(req.Token))
	if balance == nil {
		return nil, status.Error(codes.NotFound, "swap pool from provider not found")
	}
	return &pb.SwapPoolFromProviderResponse{
		AmountHub:   amount0.String(),
		AmountToken: amount1.String(),
		Balance:     balance.String(),
	}, nil
}
