package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/helpers"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strings"
)

func (s *Service) BestTrade(ctx context.Context, req *pb.BestTradeRequest) (*pb.BestTradeResponse, error) {
	amount := helpers.StringToBigIntOrNil(req.Amount)
	if amount == nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Errorf("cannot decode %s into big.Int", amount).Error())
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	var trades []*swap.Trade
	if req.Type == pb.BestTradeRequest_input {
		trades, err = cState.Swap().GetBestTradeExactIn(req.SellCoin, req.BuyCoin, amount, 10, 4)
	} else {
		trades, err = cState.Swap().GetBestTradeExactOut(req.SellCoin, req.BuyCoin, amount, 10, 4)
	}
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	res := &pb.BestTradeResponse{
		Routes: make([]*pb.BestTradeResponseRoute, 0, len(trades)),
	}

	for _, trade := range trades {
		route := &pb.BestTradeResponseRoute{
			Path:      make([]uint64, 0, len(trade.Route.Path)),
			NextPrice: big.NewFloat(0).Quo(big.NewFloat(0).SetInt(trade.NextMidPrice.Value), big.NewFloat(1e18)).Text('f', 18),
			Price:     big.NewFloat(0).Quo(big.NewFloat(0).SetInt(trade.Route.MidPrice.Value), big.NewFloat(1e18)).Text('f', 18),
		}
		if req.Type == pb.BestTradeRequest_input {
			route.Result = trade.OutputAmount.Amount.String()
		} else {
			route.Result = trade.InputAmount.Amount.String()
		}
		for _, token := range trade.Route.Path {
			route.Path = append(route.Path, token.CoinID)
		}
		res.Routes = append(res.Routes, route)
	}

	return res, nil
}

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
	liquidityID := swapper.GetID()
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
