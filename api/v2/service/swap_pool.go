package service

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) LimitOrderList(ctx context.Context, req *pb.LimitOrderListRequest) (*pb.LimitOrderListResponse, error) {
	if req.SellCoin == req.BuyCoin {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	swapper := cState.Swap().GetSwapper(types.CoinID(req.SellCoin), types.CoinID(req.BuyCoin))
	if swapper.GetID() == 0 {
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	resp := &pb.LimitOrderListResponse{Orders: make([]*pb.LimitOrderResponse, 0, req.Limit)}

	orderByIndex := swapper.OrderSellByIndex
	if !swapper.IsSorted() {
		orderByIndex = swapper.OrderBuyByIndex
	}

	for i := 0; i < int(req.Limit); i++ {
		order := orderByIndex(i)
		if order == nil {
			break
		}
		resp.Orders = append(resp.Orders, &pb.LimitOrderResponse{
			Id: uint64(order.ID()),
			CoinSell: &pb.Coin{
				Id:     uint64(order.Coin1),
				Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
			},
			CoinBuy: &pb.Coin{
				Id:     uint64(order.Coin0),
				Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
			},
			WantSell: order.WantSell.String(),
			WantBuy:  order.WantBuy.String(),
			Owner:    order.Owner.String(),
			Height:   order.Height,
		})
	}

	return resp, nil
}

func (s *Service) LimitOrderIDList(ctx context.Context, req *pb.LimitOrderListRequest) (*pb.LimitOrderIDListResponse, error) {
	if req.SellCoin == req.BuyCoin {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	swapper := cState.Swap().GetSwapper(types.CoinID(req.SellCoin), types.CoinID(req.BuyCoin))
	if swapper.GetID() == 0 {
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var ids []uint64

	orderByIndex := swapper.OrderIDSellByIndex
	if !swapper.IsSorted() {
		orderByIndex = swapper.OrderIDBuyByIndex
	}

	for i := 0; i < int(req.Limit); i++ {
		id := uint64(orderByIndex(i))
		if id == 0 {
			break
		}
		ids = append(ids, id)
	}

	return &pb.LimitOrderIDListResponse{
		Ids: ids,
	}, nil
}

func (s *Service) LimitOrder(ctx context.Context, req *pb.LimitOrderRequest) (*pb.LimitOrderResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	order := cState.Swap().GetOrder(uint32(req.OrderId))
	if order == nil {
		return nil, status.Error(codes.NotFound, "limit order not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	if order.IsBuy {
		order = order.Reverse()
	}

	return &pb.LimitOrderResponse{
		Id: uint64(order.ID()),
		CoinSell: &pb.Coin{
			Id:     uint64(order.Coin1),
			Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
		},
		CoinBuy: &pb.Coin{
			Id:     uint64(order.Coin0),
			Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
		},
		WantSell: order.WantSell.String(),
		WantBuy:  order.WantBuy.String(),
		Owner:    order.Owner.String(),
		Height:   order.Height,
	}, nil
}

func (s *Service) LimitOrders(ctx context.Context, req *pb.LimitOrdersRequest) (*pb.LimitOrdersResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	resp := &pb.LimitOrdersResponse{Orders: make([]*pb.LimitOrderResponse, 0, len(req.Ids))}
	for _, id := range req.Ids {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		order := cState.Swap().GetOrder(uint32(id))
		if order == nil {
			continue
		}

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		if order.IsBuy {
			order = order.Reverse()
		}

		resp.Orders = append(resp.Orders, &pb.LimitOrderResponse{
			Id: uint64(order.ID()),
			CoinSell: &pb.Coin{
				Id:     uint64(order.Coin1),
				Symbol: cState.Coins().GetCoin(order.Coin1).GetFullSymbol(),
			},
			CoinBuy: &pb.Coin{
				Id:     uint64(order.Coin0),
				Symbol: cState.Coins().GetCoin(order.Coin0).GetFullSymbol(),
			},
			WantSell: order.WantSell.String(),
			WantBuy:  order.WantBuy.String(),
			Owner:    order.Owner.String(),
			Height:   order.Height,
		})
	}

	return resp, nil
}

func (s *Service) SwapPool(ctx context.Context, req *pb.SwapPoolRequest) (*pb.SwapPoolResponse, error) {
	if req.Coin0 == req.Coin1 {
		return nil, status.Error(codes.InvalidArgument, "equal coins id")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
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

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	return &pb.SwapPoolResponse{
		Amount0:   reserve0.String(),
		Amount1:   reserve1.String(),
		Liquidity: cState.Coins().GetCoinBySymbol(transaction.LiquidityCoinSymbol(liquidityID), 0).Volume().String(),
	}, nil
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
		return nil, status.Error(codes.NotFound, "pair not found")
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
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
