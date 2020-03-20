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

func (s *Service) EstimateCoinBuy(_ context.Context, req *pb.EstimateCoinBuyRequest) (*pb.EstimateCoinBuyResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)

	var result *big.Int

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.FailedPrecondition, "\"From\" coin equals to \"to\" coin")
	}

	if !cState.Coins.Exists(coinToSell) {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.FailedPrecondition, "Coin to sell not exists")
	}

	if !cState.Coins.Exists(coinToBuy) {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.FailedPrecondition, "Coin to buy not exists")
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if coinToSell != types.GetBaseCoin() {
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"has":      coin.Reserve().String(),
				"required": commissionInBaseCoin.String(),
			}))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	valueToBuy, ok := big.NewInt(0).SetString(req.ValueToBuy, 10)
	if !ok {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.InvalidArgument, "Value to buy not specified")
	}

	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToBuy)
		result = formula.CalculatePurchaseAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(valueToBuy) < 0 {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), valueToBuy.String())), transaction.EncodeError(map[string]string{
				"has":      coin.Reserve().String(),
				"required": valueToBuy.String(),
			}))
		}

		result = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)
	default:
		coinFrom := cState.Coins.GetCoin(coinToSell)
		coinTo := cState.Coins.GetCoin(coinToBuy)
		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToBuy)

		if coinFrom.Reserve().Cmp(baseCoinNeeded) < 0 {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), baseCoinNeeded.String())), transaction.EncodeError(map[string]string{
				"has":      coinFrom.Reserve().String(),
				"required": baseCoinNeeded.String(),
			}))

		}

		result = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), baseCoinNeeded)
	}

	return &pb.EstimateCoinBuyResponse{
		WillPay:    result.String(),
		Commission: commission.String(),
	}, nil
}
