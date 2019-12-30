package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/MinterTeam/minter-go-node/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) CoinInfo(_ context.Context, req *pb.CoinInfoRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.CoinInfoResponse), status.Error(codes.NotFound, err.Error())
	}

	coin := cState.Coins.GetCoin(types.StrToCoinSymbol(req.Symbol))
	if coin == nil {
		return new(pb.CoinInfoResponse), status.Error(codes.FailedPrecondition, "Coin not found")
	}

	return &pb.CoinInfoResponse{
		Name:           coin.Name(),
		Symbol:         coin.Symbol().String(),
		Volume:         coin.Volume().String(),
		Crr:            fmt.Sprintf("%d", coin.Crr()),
		ReserveBalance: coin.Reserve().String(),
	}, nil
}
