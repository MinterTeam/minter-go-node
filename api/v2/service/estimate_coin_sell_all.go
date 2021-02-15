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

// EstimateCoinSellAll return estimate of sell all coin transaction.
func (s *Service) EstimateCoinSellAll(ctx context.Context, req *pb.EstimateCoinSellAllRequest) (*pb.EstimateCoinSellAllResponse, error) {
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
			return nil,
				s.createError(status.New(codes.NotFound, "Coin to sell not exists"), transaction.EncodeError(code.NewCoinNotExists("", coinToSell.String())))
		}
	}

	if coinToSell == coinToBuy {
		return nil, s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"),
			transaction.EncodeError(code.NewCrossConvert(coinToSell.String(), cState.Coins().GetCoin(coinToSell).GetFullSymbol(), coinToBuy.String(), cState.Coins().GetCoin(coinToBuy).GetFullSymbol())))
	}

	var coinFrom transaction.CalculateCoin
	coinFrom = cState.Coins().GetCoin(coinToSell)
	coinTo := cState.Coins().GetCoin(coinToBuy)

	commissions := cState.Commission().GetCommissions()

	var valueBancor, valuePool *big.Int
	var errBancor, errPool error
	value := big.NewInt(0)
	if req.SwapFrom == pb.SwapFrom_bancor || req.SwapFrom == pb.SwapFrom_optimal {
		commissionInBaseCoin := new(big.Int).Set(commissions.SellAllBancor)
		if !commissions.Coin.IsBaseCoin() {
			commissionInBaseCoin = cState.Swap().GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
		}
		if req.GasPrice > 1 {
			commissionInBaseCoin.Mul(commissionInBaseCoin, big.NewInt(int64(req.GasPrice)))
		}

		commission := commissionInBaseCoin
		if !coinFrom.ID().IsBaseCoin() {
			commissionPoolSwapper := cState.Swap().GetSwapper(coinFrom.ID(), types.GetBaseCoinID())
			commissionFrom, isFromPool, errResp := transaction.CalculateCommission(cState, commissionPoolSwapper, coinFrom, commissionInBaseCoin)
			if errResp != nil {
				return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
			}
			commission = commissionFrom
			if isFromPool == false && !coinFrom.ID().IsBaseCoin() {
				coinFrom = transaction.NewDummyCoin(
					coinFrom.ID(),
					big.NewInt(0).Sub(coinFrom.Volume(), commission),
					big.NewInt(0).Sub(coinFrom.Reserve(), commissionInBaseCoin),
					coinFrom.Crr(),
					coinFrom.GetFullSymbol(),
					coinFrom.MaxSupply(),
				)
			}
		}

		valueToSell.Sub(valueToSell, commission)

		valueBancor, errBancor = s.calcSellAllFromBancor(valueToSell, coinTo, coinFrom, commissionInBaseCoin)
	}
	if req.SwapFrom == pb.SwapFrom_pool || req.SwapFrom == pb.SwapFrom_optimal {
		commissionInBaseCoin := new(big.Int).Set(commissions.SellAllPoolBase)
		if !commissions.Coin.IsBaseCoin() {
			commissionInBaseCoin = cState.Swap().GetSwapper(types.GetBaseCoinID(), commissions.Coin).CalculateSellForBuy(commissionInBaseCoin)
		}
		if req.GasPrice > 1 {
			commissionInBaseCoin.Mul(commissionInBaseCoin, big.NewInt(int64(req.GasPrice)))
		}

		swapper := cState.Swap().GetSwapper(coinFrom.ID(), coinTo.ID())
		commission := commissionInBaseCoin
		if !coinFrom.ID().IsBaseCoin() {
			commissionPoolSwapper := cState.Swap().GetSwapper(coinFrom.ID(), types.GetBaseCoinID())
			commissionFrom, isFromPool, errResp := transaction.CalculateCommission(cState, commissionPoolSwapper, coinFrom, commissionInBaseCoin)
			if errResp != nil {
				return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
			}
			commission = commissionFrom
			if isFromPool == true && coinTo.ID().IsBaseCoin() {
				swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
			}
		}

		valueToSell.Sub(valueToSell, commission)

		valuePool, errPool = s.calcSellAllFromPool(valueToSell, swapper, coinFrom, coinTo)
	}

	swapFrom := req.SwapFrom
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
			if valueBancor.Cmp(valuePool) == -1 {
				value = valuePool
				swapFrom = pb.SwapFrom_pool
			} else {
				value = valueBancor
				swapFrom = pb.SwapFrom_bancor
			}
			break
		}

		if valueBancor != nil {
			value = valueBancor
			swapFrom = pb.SwapFrom_bancor
			break
		}
		if valuePool != nil {
			value = valuePool
			swapFrom = pb.SwapFrom_pool
			break
		}

		respBancor, _ := status.FromError(errBancor)
		respPool, _ := status.FromError(errPool)
		return nil, s.createError(status.New(codes.FailedPrecondition, "not possible to exchange"),
			transaction.EncodeError(code.NewCommissionCoinNotSufficient(respBancor.Message(), respPool.Message())))
	}

	return &pb.EstimateCoinSellAllResponse{
		WillGet:  value.String(),
		SwapFrom: swapFrom,
	}, nil
}

func (s *Service) calcSellAllFromPool(value *big.Int, swapChecker swap.EditableChecker, coinFrom transaction.CalculateCoin, coinTo *coins.Model) (*big.Int, error) {
	if !swapChecker.IsExist() {
		return nil, s.createError(status.New(codes.NotFound, fmt.Sprintf("swap pool between coins %s and %s not exists", coinFrom.GetFullSymbol(), coinTo.GetFullSymbol())), transaction.EncodeError(code.NewPairNotExists(coinFrom.ID().String(), coinTo.ID().String())))
	}
	buyValue := swapChecker.CalculateBuyForSell(value)
	if errResp := transaction.CheckSwap(swapChecker, coinFrom, coinTo, value, buyValue, false); errResp != nil {
		return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
	}
	return buyValue, nil
}

func (s *Service) calcSellAllFromBancor(value *big.Int, coinTo *coins.Model, coinFrom transaction.CalculateCoin, commissionInBaseCoin *big.Int) (*big.Int, error) {
	if !coinTo.BaseOrHasReserve() {
		return nil, s.createError(status.New(codes.FailedPrecondition, "coin to buy has no reserve"), transaction.EncodeError(code.NewCoinHasNotReserve(
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

	if !coinFrom.ID().IsBaseCoin() {
		value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if !coinTo.ID().IsBaseCoin() {
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if !coinFrom.ID().IsBaseCoin() {
		value = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), value)
		if errResp := transaction.CheckReserveUnderflow(coinFrom, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}
	value.Sub(value, commissionInBaseCoin)
	if value.Sign() != 1 {
		return nil, status.New(codes.FailedPrecondition, "Not enough coins to pay commission").Err()
	}
	if !coinTo.ID().IsBaseCoin() {
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	return value, nil
}
