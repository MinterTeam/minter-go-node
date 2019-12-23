package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

func (s *Service) EstimateCoinBuy(_ context.Context, req *pb.EstimateCoinBuyRequest) (*pb.EstimateCoinBuyResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.EstimateCoinBuyResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)

	var result *big.Int

	if coinToSell == coinToBuy {
		return &pb.EstimateCoinBuyResponse{
			Error: &pb.Error{
				Code:    "400",
				Message: "\"From\" coin equals to \"to\" coin",
			},
		}, nil
	}

	if !cState.Coins.Exists(coinToSell) {
		return &pb.EstimateCoinBuyResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin to sell not exists",
			},
		}, nil
	}

	if !cState.Coins.Exists(coinToBuy) {
		return &pb.EstimateCoinBuyResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin to buy not exists",
			},
		}, nil
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if coinToSell != types.GetBaseCoin() {
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return &pb.EstimateCoinBuyResponse{
				Error: &pb.Error{
					Code: "400",
					Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
						coin.Reserve().String(), commissionInBaseCoin.String()),
				},
			}, nil
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}
	valueToBuy, _ := big.NewInt(0).SetString(req.ValueToBuy, 10)
	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToBuy)
		result = formula.CalculatePurchaseAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(valueToBuy) < 0 {
			return &pb.EstimateCoinBuyResponse{
				Error: &pb.Error{
					Code: "400",
					Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
						coin.Reserve().String(), valueToBuy.String()),
				},
			}, nil
		}

		result = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)
	default:
		coinFrom := cState.Coins.GetCoin(coinToSell)
		coinTo := cState.Coins.GetCoin(coinToBuy)
		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToBuy)

		if coinFrom.Reserve().Cmp(baseCoinNeeded) < 0 {
			return &pb.EstimateCoinBuyResponse{
				Error: &pb.Error{
					Code: "400", Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
						coinFrom.Reserve().String(), baseCoinNeeded.String()),
				},
			}, nil
		}

		result = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), baseCoinNeeded)
	}

	return &pb.EstimateCoinBuyResponse{
		WillPay:    result.String(),
		Commission: commission.String(),
	}, nil
}
