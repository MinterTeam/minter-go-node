package service

import (
	"context"
	"encoding/hex"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

func (s *Service) SwapPool(_ context.Context, req *pb.SwapPoolRequest) (*pb.SwapPoolResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	reserve0, reserve1, liquidityID := cState.Swap().SwapPool(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	if liquidityID == 0 {
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	return &pb.SwapPoolResponse{
		Amount0:   reserve0.String(),
		Amount1:   reserve1.String(),
		Liquidity: cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(liquidityID), 0).Volume().String(),
	}, nil
}

func (s *Service) SwapPoolProvider(_ context.Context, req *pb.SwapPoolProviderRequest) (*pb.SwapPoolResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	if !strings.HasPrefix(strings.Title(req.Provider), "Mx") {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	decodeString, err := hex.DecodeString(req.Provider[2:])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}
	address := types.BytesToAddress(decodeString)

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	swapper := cState.Swap().GetSwapper(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	liquidityID := swapper.CoinID()
	if liquidityID == 0 {
		return nil, status.Error(codes.NotFound, "pair from provider not found")
	}

	liquidityCoin := cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(liquidityID), 0)
	balance := cState.Accounts().GetBalance(address, liquidityCoin.ID())

	amount0, amount1 := swapper.Amounts(balance, liquidityCoin.Volume())
	return &pb.SwapPoolResponse{
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
		Liquidity: balance.String(),
	}, nil
}
