package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state/coins"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
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

	commissions := cState.Commission().GetCommissions()
	commissionInBaseCoin := commissions.Convert
	if !commissions.Coin.IsBaseCoin() {
		commissionInBaseCoin = cState.Swap().GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
	}
	commissionPoolSwapper := cState.Swap().GetSwapper(coinFrom.ID(), types.GetBaseCoinID())
	commission, _, errResp := transaction.CalculateCommission(cState, commissionPoolSwapper, coinFrom, commissionInBaseCoin)
	if errResp != nil {
		return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
	}

	var valueBancor, valuePool *big.Int
	var errBancor, errPool error
	value := big.NewInt(0)
	if req.SwapFrom == pb.SwapFrom_bancor || req.SwapFrom == pb.SwapFrom_optimal {
		valueBancor, errBancor = s.calcBuyFromBancor(valueToBuy, coinTo, coinFrom)
	}
	if req.SwapFrom == pb.SwapFrom_pool || req.SwapFrom == pb.SwapFrom_optimal {
		valuePool, errPool = s.calcBuyFromPool(valueToBuy, cState.Swap().GetSwapper(coinFrom.ID(), coinTo.ID()), coinFrom, coinTo)
	}

	switch req.SwapFrom {
	case pb.SwapFrom_bancor:
		if errBancor != nil {
			return nil, errBancor
		}
		value = valueBancor
	case pb.SwapFrom_pool:
		if errPool != nil {
			return nil, errPool
		}
		value = valuePool
	default:
		if valueBancor != nil && valuePool != nil {
			if valueBancor.Cmp(valuePool) == 1 {
				value = valuePool
			} else {
				value = valueBancor
			}
			break
		}

		if valueBancor != nil {
			value = valueBancor
			break
		}
		if valuePool != nil {
			value = valuePool
			break
		}

		respBancor, _ := status.FromError(errBancor)
		respPool, _ := status.FromError(errPool)
		return nil, s.createError(status.New(codes.FailedPrecondition, "not possible to exchange"),
			transaction.EncodeError(code.NewCommissionCoinNotSufficient(respBancor.Message(), respPool.Message())))
	}

	return &pb.EstimateCoinBuyResponse{
		WillPay:    value.String(),
		Commission: commission.String(),
	}, nil
}

func (s *Service) calcBuyFromPool(value *big.Int, swapChecker swap.EditableChecker, coinFrom *coins.Model, coinTo *coins.Model) (*big.Int, error) {
	if !swapChecker.IsExist() {
		return nil, s.createError(status.New(codes.NotFound, fmt.Sprintf("swap pair beetwen coins %s and %s not exists in pool", coinFrom.GetFullSymbol(), coinTo.GetFullSymbol())), transaction.EncodeError(code.NewPairNotExists(coinFrom.ID().String(), coinTo.ID().String())))
	}
	if errResp := transaction.CheckSwap(swapChecker, coinFrom, coinTo, swapChecker.CalculateSellForBuy(value), value, true); errResp != nil {
		return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
	}
	return value, nil
}

func (s *Service) calcBuyFromBancor(value *big.Int, coinTo *coins.Model, coinFrom *coins.Model) (*big.Int, error) {
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

	if !coinTo.ID().IsBaseCoin() {
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
		value = formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
	}
	if !coinFrom.ID().IsBaseCoin() {
		value = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}
	return value, nil
}
