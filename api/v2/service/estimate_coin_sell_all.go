package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
)

// EstimateCoinSellAll return estimate of sell all coin transaction.
func (s *Service) EstimateCoinSellAll(ctx context.Context, req *pb.EstimateCoinSellAllRequest) (*pb.EstimateCoinSellAllResponse, error) {
	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return new(pb.EstimateCoinSellAllResponse), status.Error(codes.NotFound, err.Error())
	}

	cState.RLock()
	defer cState.RUnlock()

	gasPrice := req.GasPrice
	if gasPrice < 1 {
		gasPrice = 1
	}

	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10)
	if !ok {
		return new(pb.EstimateCoinSellAllResponse), status.Error(codes.InvalidArgument, "Value to sell not specified")
	}

	var coinToBuy types.CoinID
	if req.GetCoinToBuy() != "" {
		symbol := cState.Coins().GetCoinBySymbol(types.StrToCoinSymbol(req.GetCoinToBuy()), types.GetVersionFromSymbol(req.GetCoinToBuy()))
		if symbol == nil {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.NotFound, "Coin to buy not exists"), transaction.EncodeError(code.NewCoinNotExists(req.GetCoinToBuy(), "")))
		}
		coinToBuy = symbol.ID()
	} else {
		coinToBuy = types.CoinID(req.GetCoinIdToBuy())
		if !cState.Coins().Exists(coinToBuy) {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.NotFound, "Coin to buy not exists"), transaction.EncodeError(transaction.EncodeError(code.NewCoinNotExists("", coinToBuy.String()))))
		}
	}

	var coinToSell types.CoinID
	if req.GetCoinToSell() != "" {
		symbol := cState.Coins().GetCoinBySymbol(types.StrToCoinSymbol(req.GetCoinToSell()), types.GetVersionFromSymbol(req.GetCoinToSell()))
		if symbol == nil {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(code.NewCoinNotExists(req.GetCoinToSell(), "")))
		}
		coinToSell = symbol.ID()
	} else {
		coinToSell = types.CoinID(req.GetCoinIdToSell())
		if !cState.Coins().Exists(coinToSell) {
			return new(pb.EstimateCoinSellAllResponse),
				s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(code.NewCoinNotExists("", coinToSell.String())))
		}
	}

	if coinToSell == coinToBuy {
		return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"), transaction.EncodeError(map[string]string{
			"code":            "400",
			"coin_id_to_sell": coinToSell.String(),
			"coin_to_sell":    cState.Coins().GetCoin(coinToSell).Symbol().String(),
			"coin_id_to_buy":  coinToBuy.String(),
			"coin_to_buy":     cState.Coins().GetCoin(coinToBuy).Symbol().String(),
		}))
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	value := valueToSell
	if !coinToSell.IsBaseCoin() {
		if coinFrom.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return new(pb.EstimateCoinSellAllResponse), s.createError(
				status.New(codes.InvalidArgument, fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
					coinFrom.Reserve().String(), commissionInBaseCoin.String())),
				transaction.EncodeError(code.NewCoinReserveNotSufficient(
					coinFrom.GetFullSymbol(),
					coinFrom.ID().String(),
					coinFrom.Reserve().String(),
					commissionInBaseCoin.String(),
				)),
			)
		}

		value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if !coinToBuy.IsBaseCoin() {
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return new(pb.EstimateCoinSellAllResponse), s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	}

	return &pb.EstimateCoinSellAllResponse{
		WillGet: value.String(),
	}, nil
}
