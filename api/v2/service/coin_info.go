package service

import (
	"context"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CoinInfo returns information about coin symbol.
func (s *Service) CoinInfo(ctx context.Context, req *pb.CoinInfoRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if coinLen := len(req.Symbol); coinLen == 0 || req.Symbol[coinLen-1] == '-' {
		return nil, s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists(req.Symbol, "")))
	}
	coin := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(req.Symbol), types.GetVersionFromSymbol(req.Symbol))
	if coin == nil {
		return nil, s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists(req.Symbol, "")))
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var ownerAddress *wrapperspb.StringValue
	info := cState.Coins().GetSymbolInfo(coin.Symbol())
	if info != nil && info.OwnerAddress() != nil {
		ownerAddress = wrapperspb.String(info.OwnerAddress().String())
	}

	return &pb.CoinInfoResponse{
		Id:             uint64(coin.ID()),
		Name:           coin.Name(),
		Symbol:         coin.GetFullSymbol(),
		Volume:         coin.Volume().String(),
		Crr:            uint64(coin.Crr()),
		ReserveBalance: coin.Reserve().String(),
		MaxSupply:      coin.MaxSupply().String(),
		OwnerAddress:   ownerAddress,
		Mintable:       coin.Mintable,
		Burnable:       coin.Burnable,
	}, nil
}

// CoinInfoById returns information about coin ID.
func (s *Service) CoinInfoById(ctx context.Context, req *pb.CoinIdRequest) (*pb.CoinInfoResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	coin := cState.Coins().GetCoin(types.CoinID(req.Id))
	if coin == nil {
		return nil, s.createError(status.New(codes.NotFound, "Coin not found"), transaction.EncodeError(code.NewCoinNotExists("", strconv.Itoa(int(req.Id)))))
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var ownerAddress *wrapperspb.StringValue
	info := cState.Coins().GetSymbolInfo(coin.Symbol())
	if info != nil && info.OwnerAddress() != nil {
		ownerAddress = wrapperspb.String(info.OwnerAddress().String())
	}

	return &pb.CoinInfoResponse{
		Id:             uint64(coin.ID()),
		Name:           coin.Name(),
		Symbol:         coin.GetFullSymbol(),
		Volume:         coin.Volume().String(),
		Crr:            uint64(coin.Crr()),
		ReserveBalance: coin.Reserve().String(),
		MaxSupply:      coin.MaxSupply().String(),
		OwnerAddress:   ownerAddress,
		Mintable:       coin.Mintable,
		Burnable:       coin.Burnable,
	}, nil
}
