package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (s *Service) Pairs(context.Context, *empty.Empty) (*pb.PairsResponse, error) {
	state := s.blockchain.CurrentState()
	pairs, err := state.Swap().Pairs()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	ps := make([]*pb.PairsResponse_Pair, 0, len(pairs))
	for _, pair := range pairs {
		ps = append(ps, &pb.PairsResponse_Pair{
			Coin0: &pb.Coin{
				Id:     uint64(pair.CoinA),
				Symbol: state.Coins().GetCoin(pair.CoinA).GetFullSymbol(),
			},
			Coin1: &pb.Coin{
				Id:     uint64(pair.CoinB),
				Symbol: state.Coins().GetCoin(pair.CoinB).GetFullSymbol(),
			},
		})
	}
	return &pb.PairsResponse{
		Pairs: ps,
	}, nil
}

func (s *Service) Pair(_ context.Context, req *pb.PairRequest) (*pb.PairResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}
	state := s.blockchain.CurrentState()
	pair, err := state.Swap().Pair(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if pair != nil {
		return nil, status.Error(codes.NotFound, "pair not found")
	}
	reserve0, reserve1 := pair.Reserves()
	return &pb.PairResponse{
		Reserve0:    reserve0.String(),
		Reserve1:    reserve1.String(),
		TotalSupply: pair.TotalSupply().String(),
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
	pair, err := state.Swap().Pair(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if pair != nil {
		return nil, status.Error(codes.NotFound, "pair not found")
	}
	balance := pair.Balance(address)
	amount0, amount1 := pair.Amounts(balance)
	return &pb.PairFromProviderResponse{
		Amount0: amount0.String(),
		Amount1: amount1.String(),
		Balance: balance.String(),
	}, nil
}
