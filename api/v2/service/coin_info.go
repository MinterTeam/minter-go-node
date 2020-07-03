package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) CoinInfo(ctx context.Context, req *pb.CoinInfoRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.CoinInfoResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coin := cState.Coins().GetCoin(types.StrToCoinSymbol(req.Symbol))
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
