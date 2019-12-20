package service

import (
	"context"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

func (s *Service) EstimateCoinSellAll(_ context.Context, req *pb.EstimateCoinSellAllRequest) (*pb.EstimateCoinSellAllResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.EstimateCoinSellAllResponse{
			Error: &pb.Error{
				Data: err.Error(),
			},
		}, nil
	}

	gasPrice := req.GasPrice
	if gasPrice < 1 {
		gasPrice = 1
	}

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)
	valueToSell, _ := big.NewInt(0).SetString(req.ValueToSell, 10)

	var result *big.Int

	if coinToSell == coinToBuy {
		return &pb.EstimateCoinSellAllResponse{
			Error: &pb.Error{
				Code:    "400",
				Message: "\"From\" coin equals to \"to\" coin",
			},
		}, nil
	}

	if !cState.Coins.Exists(coinToSell) {
		return &pb.EstimateCoinSellAllResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin to sell not exists",
			},
		}, nil
	}

	if !cState.Coins.Exists(coinToBuy) {
		return &pb.EstimateCoinSellAllResponse{
			Error: &pb.Error{
				Code:    "404",
				Message: "Coin to buy not exists",
			},
		}, nil
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToBuy)

		valueToSell.Sub(valueToSell, commission)
		if valueToSell.Cmp(big.NewInt(0)) != 1 {
			return &pb.EstimateCoinSellAllResponse{
				Error: &pb.Error{
					Code:    "400",
					Message: "Not enough coins to pay commission",
				},
			}, nil
		}

		result = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToSell)
		result = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)

		result.Sub(result, commission)
		if result.Cmp(big.NewInt(0)) != 1 {
			return &pb.EstimateCoinSellAllResponse{
				Error: &pb.Error{
					Code:    "400",
					Message: "Not enough coins to pay commission",
				},
			}, nil
		}
	default:
		coinFrom := cState.Coins.GetCoin(coinToSell)
		coinTo := cState.Coins.GetCoin(coinToBuy)
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)

		basecoinValue.Sub(basecoinValue, commission)
		if basecoinValue.Cmp(big.NewInt(0)) != 1 {
			return &pb.EstimateCoinSellAllResponse{
				Error: &pb.Error{
					Code:    "400",
					Message: "Not enough coins to pay commission",
				},
			}, nil
		}

		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
	}

	return &pb.EstimateCoinSellAllResponse{
		Result: &pb.EstimateCoinSellAllResponse_Result{
			WillGet: result.String(),
		},
	}, nil
}
