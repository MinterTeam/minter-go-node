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

func (s *Service) EstimateCoinSell(_ context.Context, req *pb.EstimateCoinSellRequest) (*pb.EstimateCoinSellResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.EstimateCoinSellResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)

	var result *big.Int

	if coinToSell == coinToBuy {
		return &pb.EstimateCoinSellResponse{
			Error: &pb.Error{
				Code:    "400",
				Message: "\"From\" coin equals to \"to\" coin",
			},
		}, nil
	}

	if !cState.Coins.Exists(coinToSell) {
		return &pb.EstimateCoinSellResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin to sell not exists",
			},
		}, nil
	}

	if !cState.Coins.Exists(coinToBuy) {
		return &pb.EstimateCoinSellResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin to buy not exists",
			},
		}, nil
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)
	valueToSell, _ := big.NewInt(0).SetString(req.ValueToSell, 10)

	if coinToSell != types.GetBaseCoin() {
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return &pb.EstimateCoinSellResponse{
				Error: &pb.Error{
					Code: "400",
					Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
						coin.Reserve().String(), commissionInBaseCoin.String()),
				},
			}, nil
		}

		if coin.Volume().Cmp(valueToSell) < 0 {
			return &pb.EstimateCoinSellResponse{
				Error: &pb.Error{
					Code: "400",
					Message: fmt.Sprintf("Coin volume is not sufficient for transaction. Has: %s, required %s",
						coin.Volume().String(), valueToSell.String()),
				},
			}, nil
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToBuy)
		result = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToSell)
		result = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	default:
		coinFrom := cState.Coins.GetCoin(coinToSell)
		coinTo := cState.Coins.GetCoin(coinToBuy)
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
	}

	return &pb.EstimateCoinSellResponse{
		WillGet:    result.String(),
		Commission: commission.String(),
	}, nil
}
