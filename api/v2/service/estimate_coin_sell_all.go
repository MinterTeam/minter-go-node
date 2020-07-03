package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) EstimateCoinSellAll(ctx context.Context, req *pb.EstimateCoinSellAllRequest) (*pb.EstimateCoinSellAllResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateCoinSellAllResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	gasPrice := req.GasPrice
	if gasPrice < 1 {
		gasPrice = 1
	}

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)
	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10) //todo: mb delete?
	if !ok {
		return new(pb.EstimateCoinSellAllResponse), status.Error(codes.InvalidArgument, "Value to sell not specified")
	}

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
			"coin_to_buy":  coinToBuy.String(),
		}))
	}

	if !cState.Coins().Exists(coinToSell) {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "Coin to sell not exists"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
		}))
	}

	if !cState.Coins().Exists(coinToBuy) {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "Coin to buy not exists"), transaction.EncodeError(map[string]string{
			"coin_to_buy": coinToBuy.String(),
		}))
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	value := valueToSell
	if coinToSell != types.GetBaseCoin() {
		value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if coinToBuy != types.GetBaseCoin() {
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo.Volume(), value, coinTo.MaxSupply()); errResp != nil {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	}

	return &pb.EstimateCoinSellAllResponse{
		WillGet: value.String(),
	}, nil
}
