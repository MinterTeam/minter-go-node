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

// EstimateCoinSell return estimate of sell coin transaction.
func (s *Service) EstimateCoinSell(ctx context.Context, req *pb.EstimateCoinSellRequest) (*pb.EstimateCoinSellResponse, error) {
	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "Value to sell not specified")
	}

	cState, err := s.blockchain.GetStateForHeight(req.Height)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	var coinToBuy types.CoinID
	if req.GetCoinToBuy() != "" {
		symbol := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(req.GetCoinToBuy()), types.GetVersionFromSymbol(req.GetCoinToBuy()))
		if symbol == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin to buy not exists"), transaction.EncodeError(code.NewCoinNotExists(req.GetCoinToBuy(), "")))
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
	commissionPoolSwapper := cState.Swap().GetSwapper(coinFrom.ID(), types.GetBaseCoinID())
	commission, _, errResp := transaction.CalculateCommission(cState, commissionPoolSwapper, coinFrom, commissionInBaseCoin)
	if errResp != nil {
		return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
	}

	value := valueToSell
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

		if !coinToSell.IsBaseCoin() {
			value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
			if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
				return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
			}
		}

		if !coinToBuy.IsBaseCoin() {
			value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
			if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
				return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
			}
		}
	} else {
		var err error
		value, err = cState.Swap().PairCalculateBuyForSell(coinFrom.ID(), coinTo.ID(), valueToSell)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		commissionPoolSwapper := cState.Swap().GetSwapper(coinFrom.ID(), coinTo.ID())
		if errResp := transaction.CheckSwap(commissionPoolSwapper, coinFrom, coinTo, valueToSell, value, false); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	res := &pb.EstimateCoinSellResponse{
		WillGet:    value.String(),
		Commission: commission.String(),
	}
	return res, nil
}
