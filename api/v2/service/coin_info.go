package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CoinInfo returns information about coin symbol.
func (s *Service) CoinInfo(ctx context.Context, req *pb.CoinInfoRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.CoinInfoResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coin := cState.Coins().GetCoinBySymbol(types.StrToCoinSymbol(req.Symbol), types.GetVersionFromSymbol(req.Symbol))
	if coin == nil {
		return new(pb.CoinInfoResponse), s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists(req.Symbol, "")))
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.CoinInfoResponse), timeoutStatus.Err()
	}

	var ownerAddress *wrappers.StringValue
	info := cState.Coins().GetSymbolInfo(coin.Symbol())
	if info != nil && info.OwnerAddress() != nil {
		ownerAddress = &wrappers.StringValue{
			Value: info.OwnerAddress().String(),
		}
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

// Returns information about coin ID.
func (s *Service) CoinInfoById(ctx context.Context, req *pb.CoinIdRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.CoinInfoResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coin := cState.Coins().GetCoin(types.CoinID(req.Id))
	if coin == nil {
		return new(pb.CoinInfoResponse), s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists("", strconv.Itoa(int(req.Id)))))
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return new(pb.CoinInfoResponse), timeoutStatus.Err()
	}

	var ownerAddress *wrappers.StringValue
	info := cState.Coins().GetSymbolInfo(coin.Symbol())
	if info != nil && info.OwnerAddress() != nil {
		ownerAddress = &wrappers.StringValue{
			Value: info.OwnerAddress().String(),
		}
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
