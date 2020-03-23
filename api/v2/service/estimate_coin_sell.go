package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) EstimateCoinSell(_ context.Context, req *pb.EstimateCoinSellRequest) (*pb.EstimateCoinSellResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateCoinSellResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
			"coin_to_buy":  coinToBuy.String(),
		}))
	}

	if !cState.Coins.Exists(coinToSell) {
		return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, "Coin to sell not exists"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
		}))
	}

	if !cState.Coins.Exists(coinToBuy) {
		return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, "Coin to buy not exists"), transaction.EncodeError(map[string]string{
			"coin_to_buy": coinToSell.String(),
		}))

	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)
	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10)
	if !ok {
		return new(pb.EstimateCoinSellResponse), status.Error(codes.InvalidArgument, "Value to sell not specified")
	}

	if coinToSell != types.GetBaseCoin() {
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"has":      coin.Reserve().String(),
				"required": commissionInBaseCoin.String(),
			}))
		}

		if coin.Volume().Cmp(valueToSell) < 0 {
			return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), valueToSell.String())), transaction.EncodeError(map[string]string{
				"has":      coin.Reserve().String(),
				"required": valueToSell.String(),
			}))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	var result = big.NewInt(0)
	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToBuy)
		result.Set(formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell))
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToSell)
		result.Set(formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell))
	default:
		coinFrom := cState.Coins.GetCoin(coinToSell)
		coinTo := cState.Coins.GetCoin(coinToBuy)
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		result.Set(formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue))
	}

	return &pb.EstimateCoinSellResponse{
		WillGet:    result.String(),
		Commission: commission.String(),
	}, nil
}
