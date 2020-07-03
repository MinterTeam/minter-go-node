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

func (s *Service) EstimateCoinSell(ctx context.Context, req *pb.EstimateCoinSellRequest) (*pb.EstimateCoinSellResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
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

	if !cState.Coins().Exists(coinToSell) {
		return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, "Coin to sell not exists"), transaction.EncodeError(map[string]string{
			"coin_to_sell": coinToSell.String(),
		}))
	}

	if !cState.Coins().Exists(coinToBuy) {
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

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	if coinToSell != types.GetBaseCoin() {

		if coinFrom.Volume().Cmp(valueToSell) < 0 {
			return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin volume balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Volume().String(), valueToSell.String())), transaction.EncodeError(map[string]string{
				"volume_has":      coinFrom.Volume().String(),
				"volume_required": valueToSell.String(),
			}))
		}

		if coinFrom.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), commissionInBaseCoin.String())), transaction.EncodeError(map[string]string{
				"reserve_has":      coinFrom.Reserve().String(),
				"reserve_required": commissionInBaseCoin.String(),
			}))
		}

		commission = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), commissionInBaseCoin)
	}

	value := valueToSell
	if coinToSell != types.GetBaseCoin() {
		value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if coinToBuy != types.GetBaseCoin() {
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo.Volume(), value, coinTo.MaxSupply()); errResp != nil {
			return new(pb.EstimateCoinSellResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	}

	res := &pb.EstimateCoinSellResponse{
		WillGet:    value.String(),
		Commission: commission.String(),
	}
	return res, nil
}
