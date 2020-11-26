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
			X: &pb.Coin{
				Id:     uint64(pair.XCoin),
				Symbol: state.Coins().GetCoin(pair.XCoin).GetFullSymbol(),
			},
			Y: &pb.Coin{
				Id:     uint64(pair.YCoin),
				Symbol: state.Coins().GetCoin(pair.YCoin).GetFullSymbol(),
			},
		})
	}
	return &pb.PairsResponse{
		Pairs: ps,
	}, nil
}

func (s *Service) Pair(_ context.Context, req *pb.PairRequest) (*pb.PairResponse, error) {
	if req.Y == req.X {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}
	state := s.blockchain.CurrentState()
	xVolume, yVolume, stakes, err := state.Swap().Pair(types.CoinID(req.X), types.CoinID(req.Y))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if stakes == nil {
		return nil, status.Error(codes.NotFound, "pair not found")
	}
	return &pb.PairResponse{
		XVolume: xVolume.String(),
		YVolume: yVolume.String(),
		Stakes:  stakes.String(),
	}, nil
}

func (s *Service) PairFromProvider(_ context.Context, req *pb.PairFromProviderRequest) (*pb.PairFromProviderResponse, error) {
	if !strings.HasPrefix(strings.Title(req.Provider), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	if req.Y == req.X {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	decodeString, err := hex.DecodeString(req.Provider[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	address := types.BytesToAddress(decodeString)

	state := s.blockchain.CurrentState()
	xVolume, yVolume, stake, err := state.Swap().Balance(address, types.CoinID(req.X), types.CoinID(req.Y))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if stake == nil {
		return nil, status.Error(codes.NotFound, "pair from provider not found")
	}

	return &pb.PairFromProviderResponse{
		XVolume: xVolume.String(),
		YVolume: yVolume.String(),
		Stake:   stake.String(),
	}, nil
}
