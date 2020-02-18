package service

import (
	"context"
	"fmt"
	pb "github.com/MinterTeam/minter-go-node/api/v2/api_pb"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) EstimateCoinSellAll(_ context.Context, req *pb.EstimateCoinSellAllRequest) (*pb.EstimateCoinSellAllResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateCoinSellAllResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.Lock()
	defer cState.Unlock()

	gasPrice := req.GasPrice
	if gasPrice < 1 {
		gasPrice = 1
	}

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)
	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10)
	if !ok {
		return new(pb.EstimateCoinSellAllResponse), status.Error(codes.InvalidArgument, "Value to sell not specified")
	}

	var result *big.Int

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
			"coin_to_buy":  coinToBuy.String(),
		}))
	}

	if !cState.Coins.Exists(coinToSell) {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "Coin to sell not exists"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
		}))
	}

	if !cState.Coins.Exists(coinToBuy) {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "Coin to buy not exists"), transaction.EncodeError(map[string]string{
			"coin_to_buy": coinToBuy.String(),
		}))
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToBuy)

		valueToSell.Sub(valueToSell, commission)
		if valueToSell.Cmp(big.NewInt(0)) != 1 {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "Not enough coins to pay commission"), transaction.EncodeError(map[string]string{
				"value_to_sell": valueToSell.String(),
				"coin_to_sell":  coinToSell.String(),
				"commission":    commission.String(),
			}))
		}

		result = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins.GetCoin(coinToSell)
		result = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)

		result.Sub(result, commission)
		if result.Cmp(big.NewInt(0)) != 1 {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "Not enough coins to pay commission"), transaction.EncodeError(map[string]string{
				"value_to_sell":        valueToSell.String(),
				"coin_to_sell":         coinToSell.String(),
				"coin_reserve_to_sell": coin.Reserve().String(),
				"coin_crr_to_sell":     fmt.Sprintf("%d", coin.Crr()),
				"result":               result.String(),
				"commission":           commission.String(),
			}))
		}
	default:
		coinFrom := cState.Coins.GetCoin(coinToSell)
		coinTo := cState.Coins.GetCoin(coinToBuy)
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)

		basecoinValue.Sub(basecoinValue, commission)
		if basecoinValue.Cmp(big.NewInt(0)) != 1 {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.FailedPrecondition, "Not enough coins to pay commission"), transaction.EncodeError(map[string]string{
				"coin_to_sell":         coinToSell.String(),
				"coin_to_buy":          coinToBuy.String(),
				"coin_to_sell_crr":     fmt.Sprintf("%d", coinFrom.Crr()),
				"coin_to_sell_reserve": coinFrom.Reserve().String(),
				"result":               basecoinValue.String(),
				"commission":           commission.String(),
			}))
		}

		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
	}

	return &pb.EstimateCoinSellAllResponse{
		WillGet: result.String(),
	}, nil
}
