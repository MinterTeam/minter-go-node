package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/pb"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

func (s *Service) EstimateCoinSell(_ context.Context, req *pb.EstimateCoinSellRequest) (*pb.EstimateCoinSellResponse, error) {
	cState, err := s.getStateForHeight(req.Height)
	if err != nil {
		return &pb.EstimateCoinSellResponse{}, status.Error(codes.NotFound, err.Error())
	}

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)

	var result *big.Int

	detailCoin := encodeError(map[string]string{
		"coin": coinToSell.String(),
	})

	if coinToSell == coinToBuy {
		return &pb.EstimateCoinSellResponse{}, s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"), detailCoin)
	}

	if !cState.Coins.Exists(coinToSell) {
		return &pb.EstimateCoinSellResponse{}, s.createError(status.New(codes.InvalidArgument, "Coin to sell not exists"), detailCoin)
	}

	if !cState.Coins.Exists(coinToBuy) {
		return &pb.EstimateCoinSellResponse{}, s.createError(status.New(codes.InvalidArgument, "Coin to buy not exists"), detailCoin)

	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)
	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10)
	if !ok {
		return &pb.EstimateCoinSellResponse{}, s.createError(status.New(codes.InvalidArgument, "Value to sell not specified"), "")
	}

	if coinToSell != types.GetBaseCoin() {
		coin := cState.Coins.GetCoin(coinToSell)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			bytes := encodeError(map[string]string{
				"has":      coin.Reserve().String(),
				"required": commissionInBaseCoin.String(),
			})
			return &pb.EstimateCoinSellResponse{}, s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())), bytes)
		}

		if coin.Volume().Cmp(valueToSell) < 0 {
			bytes := encodeError(map[string]string{
				"has":      coin.Reserve().String(),
				"required": valueToSell.String(),
			})
			return &pb.EstimateCoinSellResponse{}, s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), valueToSell.String())), bytes)
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
