package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) CoinId(ctx context.Context, req *pb.CoinIdRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.CoinInfoResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coin := cState.Coins().GetCoin(types.CoinID(req.Id))
	if coin == nil {
		return new(pb.CoinInfoResponse), status.Error(codes.FailedPrecondition, "Coin not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.CoinInfoResponse), timeoutStatus.Err()
	}

	var ownerAddress string
	info := cState.Coins().GetSymbolInfo(coin.Symbol())
	if info != nil && info.OwnerAddress() != nil {
		ownerAddress = info.OwnerAddress().String()
	}

	return &pb.CoinInfoResponse{
		Id:             coin.ID().String(),
		Name:           coin.Name(),
		Symbol:         coin.Symbol().String(),
		Volume:         coin.Volume().String(),
		Crr:            fmt.Sprintf("%d", coin.Crr()),
		ReserveBalance: coin.Reserve().String(),
		MaxSupply:      coin.MaxSupply().String(),
		OwnerAddress:   ownerAddress,
	}, nil
}
