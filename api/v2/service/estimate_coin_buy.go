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

func (s *Service) EstimateCoinBuy(ctx context.Context, req *pb.EstimateCoinBuyRequest) (*pb.EstimateCoinBuyResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	coinToSell := types.StrToCoinSymbol(req.CoinToSell)
	coinToBuy := types.StrToCoinSymbol(req.CoinToBuy)

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.FailedPrecondition, "\"From\" coin equals to \"to\" coin")
	}

	if !cState.Coins().Exists(coinToSell) {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.FailedPrecondition, "Coin to sell not exists")
	}

	if !cState.Coins().Exists(coinToBuy) {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.FailedPrecondition, "Coin to buy not exists")
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	if coinToSell != types.GetBaseCoin() {
		if coinFrom.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"has":      coinFrom.Reserve().String(),
				"required": commissionInBaseCoin.String(),
			}))
		}

		commission = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), commissionInBaseCoin)
	}

	valueToBuy, ok := big.NewInt(0).SetString(req.ValueToBuy, 10)
	if !ok {
		return new(pb.EstimateCoinBuyResponse), status.Error(codes.InvalidArgument, "Value to buy not specified")
	}

	value := valueToBuy
	if coinToSell != types.GetBaseCoin() {
		value = formula.CalculatePurchaseAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToBuy)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if coinToBuy != types.GetBaseCoin() {
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo.Volume(), value, coinTo.MaxSupply()); errResp != nil {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
		value = formula.CalculateSaleAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	}

	return &pb.EstimateCoinBuyResponse{
		WillPay:    value.String(),
		Commission: commission.String(),
	}, nil
}
