package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/types"
)

func (s *Service) CoinInfo(_ context.Context, req *pb.CoinInfoRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.CoinInfoResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	coin := cState.Coins.GetCoin(types.StrToCoinSymbol(req.Symbol))
	if coin == nil {
		return &pb.CoinInfoResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin not found",
			},
		}, nil
	}

	return &pb.CoinInfoResponse{
		Result: &pb.CoinInfoResponse_Result{
			Name:           coin.Name(),
			Symbol:         coin.Symbol().String(),
			Volume:         coin.Volume().String(),
			Crr:            fmt.Sprintf("%d", coin.Crr()),
			ReserveBalance: coin.Reserve().String(),
		},
	}, nil
}
