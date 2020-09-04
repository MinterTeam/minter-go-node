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

	coinToSell := types.CoinID(req.CoinIdToSell)
	coinToBuy := types.CoinID(req.CoinIdToBuy)

	if !cState.Coins().Exists(coinToSell) {
		return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(map[string]string{
			"code":            "404",
			"coin_id_to_sell": coinToSell.String(),
		}))
	}

	if !cState.Coins().Exists(coinToBuy) {
		return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.NotFound, "Coin to buy not exists"), transaction.EncodeError(map[string]string{
			"code":           "404",
			"coin_id_to_buy": coinToBuy.String(),
		}))
	}

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"), transaction.EncodeError(map[string]string{
			"code":            "400",
			"coin_id_to_sell": coinToSell.String(),
			"coin_to_sell":    cState.Coins().GetCoin(coinToSell).Symbol().String(),
			"coin_id_to_buy":  coinToBuy.String(),
			"coin_to_buy":     cState.Coins().GetCoin(coinToBuy).Symbol().String(),
		}))
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	if !coinToSell.IsBaseCoin() {
		if coinFrom.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"code":     "400",
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
	if !coinToSell.IsBaseCoin() {
		value = formula.CalculatePurchaseAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToBuy)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return new(pb.EstimateCoinBuyResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if !coinToBuy.IsBaseCoin() {
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
