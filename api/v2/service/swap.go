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

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var trades []*swap.Trade
	if req.Type == pb.BestTradeRequest_input {
		trades, err = cState.Swap().GetBestTradeExactIn(req.BuyCoin, req.SellCoin, amount, 1, 4)
	} else {
		trades, err = cState.Swap().GetBestTradeExactOut(req.BuyCoin, req.SellCoin, amount, 1, 4)
	}
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	if len(trades) == 0 {
		return nil, status.Error(codes.NotFound, "route path not found")
	}

	trade := trades[0]
	route := &pb.BestTradeResponse{
		Path:  make([]uint64, 0, len(trade.Route.Path)),
		Price: big.NewFloat(0).Quo(big.NewFloat(0).SetInt(trade.Route.MidPrice.Value), big.NewFloat(1e18)).Text('f', 18),
	}
	if req.Type == pb.BestTradeRequest_input {
		route.Result = trade.OutputAmount.Amount.String()
	} else {
		route.Result = trade.InputAmount.Amount.String()
	}
	for _, token := range trade.Route.Path {
		route.Path = append(route.Path, token.CoinID)
	}

	return route, nil
}

func (s *Service) SwapPool(ctx context.Context, req *pb.SwapPoolRequest) (*pb.SwapPoolResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
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

func (s *Service) SwapPools(ctx context.Context, req *pb.SwapPoolsRequest) (*pb.SwapPoolsResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	pools := cState.Swap().SwapPools()

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	res := &pb.SwapPoolsResponse{}
	for _, pool := range pools {
		res.Pools = append(res.Pools, &pb.SwapPoolResponse{
			Amount0:   pool.Reserve0,
			Amount1:   pool.Reserve0,
			Liquidity: cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(uint32(pool.ID)), 0).Volume().String(),
		})
	}

	return res, nil
}

func (s *Service) SwapPoolProvider(ctx context.Context, req *pb.SwapPoolProviderRequest) (*pb.SwapPoolResponse, error) {
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

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	swapper := cState.Swap().GetSwapper(types.CoinID(req.Coin0), types.CoinID(req.Coin1))
	liquidityID := swapper.GetID()
	if liquidityID == 0 {
		return nil, status.Error(codes.NotFound, "pair from provider not found")
	}

	liquidityCoin := cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(liquidityID), 0)
	balance := cState.Accounts().GetBalance(address, liquidityCoin.ID())

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	amount0, amount1 := swapper.Amounts(balance, liquidityCoin.Volume())
	return &pb.SwapPoolResponse{
		Amount0:   amount0.String(),
		Amount1:   amount1.String(),
		Liquidity: balance.String(),
	}, nil
}
