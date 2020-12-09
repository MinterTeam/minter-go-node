package service

import (
	"context"
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

// EstimateCoinBuy return estimate of buy coin transaction.
func (s *Service) EstimateCoinBuy(ctx context.Context, req *pb.EstimateCoinBuyRequest) (*pb.EstimateCoinBuyResponse, error) {
	valueToBuy, ok := big.NewInt(0).SetString(req.ValueToBuy, 10)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "Value to buy not specified")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	var coinToBuy types.CoinID
	if req.GetCoinToBuy() != "" {
		symbol := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(req.GetCoinToBuy()), types.GetVersionFromSymbol(req.GetCoinToBuy()))
		if symbol == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(code.NewCoinNotExists(req.GetCoinToBuy(), "")))
		}
		coinToBuy = symbol.ID()
	} else {
		coinToBuy = types.CoinID(req.GetCoinIdToBuy())
		if !cState.Coins().Exists(coinToBuy) {
			return nil, s.createError(status.New(codes.NotFound, "Coin to buy not exists"), transaction.EncodeError(code.NewCoinNotExists("", coinToBuy.String())))
		}
	}

	var coinToSell types.CoinID
	if req.GetCoinToSell() != "" {
		symbol := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(req.GetCoinToSell()), types.GetVersionFromSymbol(req.GetCoinToSell()))
		if symbol == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(code.NewCoinNotExists(req.GetCoinToSell(), "")))
		}
		coinToSell = symbol.ID()
	} else {
		coinToSell = types.CoinID(req.GetCoinIdToSell())
		if !cState.Coins().Exists(coinToSell) {
			return nil, s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(code.NewCoinNotExists("", coinToSell.String())))
		}
	}

	if coinToSell == coinToBuy {
		return nil, s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"),
			transaction.EncodeError(code.NewCrossConvert(coinToSell.String(), cState.Coins().GetCoin(coinToSell).GetFullSymbol(), coinToBuy.String(), cState.Coins().GetCoin(coinToBuy).GetFullSymbol())))
	}

	coinFrom := cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	commissionInBaseCoin := big.NewInt(0).Mul(big.NewInt(commissions.ConvertTx), transaction.CommissionMultiplier)
	commission, _, errResp := transaction.CalculateCommission(cState, coinFrom, commissionInBaseCoin)
	if errResp != nil {
		return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
	}

	value := valueToBuy
	if !req.FromPool {
		if !coinTo.BaseOrHasReserve() {
			return nil, s.createError(status.New(codes.FailedPrecondition, "buy coin has not reserve"), transaction.EncodeError(code.NewCoinHasNotReserve(
				coinTo.GetFullSymbol(),
				coinTo.ID().String(),
			)))
		}
		if !coinFrom.BaseOrHasReserve() {
			return nil, s.createError(status.New(codes.FailedPrecondition, "sell coin has not reserve"), transaction.EncodeError(code.NewCoinHasNotReserve(
				coinFrom.GetFullSymbol(),
				coinFrom.ID().String(),
			)))
		}

		if !coinToBuy.IsBaseCoin() {
			if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, valueToBuy); errResp != nil {
				return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
			}
			value = formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToBuy)
		}
		if !coinToSell.IsBaseCoin() {
			value = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
			if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
				return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
			}
		}
	} else {
		var err error
		value, err = cState.Swap().PairCalculateSellForBuy(coinFrom.ID(), coinTo.ID(), valueToBuy)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		if err = cState.Swap().CheckSwap(coinFrom.ID(), coinTo.ID(), valueToBuy, value); err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}
	return &pb.EstimateCoinBuyResponse{
		WillPay:    value.String(),
		Commission: commission.String(),
	}, nil
}
