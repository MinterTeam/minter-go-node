package service

import (
	"context"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
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

	if len(req.Route) > 3 {
		return nil, s.createError(status.New(codes.OutOfRange, "maximum allowed length of the exchange chain is 5"), transaction.EncodeError(code.NewCustomCode(code.TooLongSwapRoute)))
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

	if coinToSell == coinToBuy && len(req.Route) == 0 {
		return nil, s.createError(status.New(codes.InvalidArgument, "\"From\" coin equals to \"to\" coin"),
			transaction.EncodeError(code.NewCrossConvert(coinToSell.String(), cState.Coins().GetCoin(coinToSell).GetFullSymbol(), coinToBuy.String(), cState.Coins().GetCoin(coinToBuy).GetFullSymbol())))
	}

	var requestCoinCommissionID types.CoinID
	if req.GetCoinCommission() != "" {
		symbol := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(req.GetCoinCommission()), types.GetVersionFromSymbol(req.GetCoinCommission()))
		if symbol == nil {
			return nil, s.createError(status.New(codes.NotFound, "Coin to pay commission not exists"), transaction.EncodeError(code.NewCoinNotExists(req.GetCoinCommission(), "")))
		}
		requestCoinCommissionID = symbol.ID()
	} else {
		requestCoinCommissionID = types.CoinID(req.GetCoinIdCommission())
		if !cState.Coins().Exists(requestCoinCommissionID) {
			return nil, s.createError(status.New(codes.NotFound, "Coin to pay commission not exists"), transaction.EncodeError(code.NewCoinNotExists("", requestCoinCommissionID.String())))
		}
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, timeoutStatus.Err()
	}

	var coinFrom, coinTo transaction.CalculateCoin
	coinFrom = cState.Coins().GetCoin(coinToSell)
	coinTo = cState.Coins().GetCoin(coinToBuy)

	value := big.NewInt(0)
	commissions := cState.Commission().GetCommissions()
	var resultCommission *big.Int
	swapFrom := req.SwapFrom

	reverseCoinIds(req.Route)

	switch req.SwapFrom {
	case pb.SwapFrom_bancor:
		commissionBancor, valueBancor, err := s.calcBuyBancorWithCommission(commissions, cState, requestCoinCommissionID, coinTo, coinFrom, valueToBuy)
		if err != nil {
			return nil, err
		}
		value = valueBancor
		resultCommission = commissionBancor
	case pb.SwapFrom_pool:
		commissionPool, valuePool, err := s.calcBuyPoolWithCommission(ctx, commissions, cState, requestCoinCommissionID, valueToBuy, coinFrom, coinTo, req.Route)
		if err != nil {
			return nil, err
		}
		value = valuePool
		resultCommission = commissionPool
	default:
		commissionBancor, valueBancor, errBancor := s.calcBuyBancorWithCommission(commissions, cState, requestCoinCommissionID, coinTo, coinFrom, valueToBuy)
		commissionPool, valuePool, errPool := s.calcBuyPoolWithCommission(ctx, commissions, cState, requestCoinCommissionID, valueToBuy, coinFrom, coinTo, req.Route)

		if valueBancor != nil && valuePool != nil {
			if valueBancor.Cmp(valuePool) == 1 {
				value = valuePool
				resultCommission = commissionPool
				swapFrom = pb.SwapFrom_pool
			} else {
				value = valueBancor
				resultCommission = commissionBancor
				swapFrom = pb.SwapFrom_bancor
			}
			break
		}

		if valueBancor != nil {
			value = valueBancor
			resultCommission = commissionBancor
			swapFrom = pb.SwapFrom_bancor
			break
		}
		if valuePool != nil {
			value = valuePool
			resultCommission = commissionPool
			swapFrom = pb.SwapFrom_pool
			break
		}

		respBancor, _ := status.FromError(errBancor)
		respPool, _ := status.FromError(errPool)
		return nil, s.createError(status.New(codes.FailedPrecondition, "not possible to exchange"),
			transaction.EncodeError(code.NewCommissionCoinNotSufficient(respBancor.Message(), respPool.Message())))
	}

	return &pb.EstimateCoinBuyResponse{
		WillPay:    value.String(),
		Commission: resultCommission.String(),
		SwapFrom:   swapFrom,
	}, nil
}

func (s *Service) calcBuyFromPool(ctx context.Context, value *big.Int, cState *state.CheckState, coinFrom transaction.CalculateCoin, coinTo transaction.CalculateCoin, route []uint64, commissionPoolSwapper swap.EditableChecker) (*big.Int, error) {
	buyCoinID := coinTo.ID()
	buyValue := big.NewInt(0).Set(value)
	coinBuy := coinTo
	for _, sellCoinInt := range append(route, uint64(coinFrom.ID())) {

		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}

		sellCoinID := types.CoinID(sellCoinInt)
		swapChecker := cState.Swap().GetSwapper(sellCoinID, buyCoinID)

		if !swapChecker.IsExist() {
			return nil, s.createError(status.New(codes.NotFound, fmt.Sprintf("swap pair beetwen coins %s and %s not exists", coinFrom.GetFullSymbol(), coinBuy.GetFullSymbol())), transaction.EncodeError(code.NewPairNotExists(coinFrom.ID().String(), coinBuy.ID().String())))
		}

		if swapChecker.GetID() == commissionPoolSwapper.GetID() {
			if sellCoinID != types.GetBaseCoinID() {
				swapChecker = commissionPoolSwapper
			} else {
				swapChecker = commissionPoolSwapper.Revert()
			}
		}

		sellValue := swapChecker.CalculateSellForBuy(buyValue)
		if sellValue == nil {
			reserve0, reserve1 := swapChecker.Reserves()
			symbolOut := coinBuy.GetFullSymbol()
			return nil, s.createError(status.New(codes.FailedPrecondition, fmt.Sprintf("You wanted to buy %s %s, but pool reserve has only %s %s", value, symbolOut, reserve1.String(), symbolOut)), transaction.EncodeError(code.NewInsufficientLiquidity(coinFrom.ID().String(), sellValue.String(), coinBuy.ID().String(), value.String(), reserve0.String(), reserve1.String())))
		}

		coinSell := coinFrom
		if sellCoinID != coinSell.ID() {
			coinSell = cState.Coins().GetCoin(sellCoinID)
		}

		if errResp := transaction.CheckSwap(swapChecker, coinSell, coinBuy, sellValue, buyValue, true); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}

		buyValue = sellValue
		coinBuy = coinSell
		buyCoinID = sellCoinID
	}

	return buyValue, nil
}

func (s *Service) calcBuyFromBancor(value *big.Int, coinTo transaction.CalculateCoin, coinFrom transaction.CalculateCoin) (*big.Int, error) {
	if !coinTo.BaseOrHasReserve() {
		return nil, s.createError(status.New(codes.FailedPrecondition, "coin to buy has no reserve"), transaction.EncodeError(code.NewCoinHasNotReserve(
			coinTo.GetFullSymbol(),
			coinTo.ID().String(),
		)))
	}
	if !coinFrom.BaseOrHasReserve() {
		return nil, s.createError(status.New(codes.FailedPrecondition, "sell coin has no reserve"), transaction.EncodeError(code.NewCoinHasNotReserve(
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

func (s *Service) calcBuyBancorWithCommission(commissions *commission.Price, cState *state.CheckState, requestCoinCommissionID types.CoinID, coinTo transaction.CalculateCoin, coinFrom transaction.CalculateCoin, valueToBuy *big.Int) (*big.Int, *big.Int, error) {
	commissionInBaseCoin := commissions.BuyBancor
	commission, commissionFromPool, err := s.commissionInCoin(cState, requestCoinCommissionID, commissions.Coin, commissionInBaseCoin)
	if err != nil {
		return nil, nil, err
	}

	if !commissionFromPool {
		if requestCoinCommissionID == coinTo.ID() {
			coinTo = transaction.NewDummyCoin(
				coinTo.ID(),
				big.NewInt(0).Sub(coinTo.Volume(), commission),
				big.NewInt(0).Sub(coinTo.Reserve(), commissionInBaseCoin),
				coinTo.Crr(),
				coinTo.GetFullSymbol(),
				coinTo.MaxSupply(),
			)
		} else if requestCoinCommissionID == coinFrom.ID() {
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

	valueBancor, errBancor := s.calcBuyFromBancor(valueToBuy, coinTo, coinFrom)
	if errBancor != nil {
		return nil, nil, errBancor
	}
	return commission, valueBancor, nil
}

func (s *Service) calcBuyPoolWithCommission(ctx context.Context, commissions *commission.Price, cState *state.CheckState, requestCoinCommissionID types.CoinID, valueToBuy *big.Int, coinFrom transaction.CalculateCoin, coinTo transaction.CalculateCoin, route []uint64) (*big.Int, *big.Int, error) {
	commissionInBaseCoin := big.NewInt(0).Add(commissions.BuyPoolBase, big.NewInt(0).Mul(commissions.BuyPoolDelta, big.NewInt(int64(len(route)))))
	commission, commissionFromPool, err := s.commissionInCoin(cState, requestCoinCommissionID, commissions.Coin, commissionInBaseCoin)
	if err != nil {
		return nil, nil, err
	}

	commissionPoolSwapper := cState.Swap().GetSwapper(requestCoinCommissionID, types.GetBaseCoinID())
	if commissionFromPool {
		commissionPoolSwapper = commissionPoolSwapper.AddLastSwapStep(commission, commissionInBaseCoin)
	}

	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, nil, timeoutStatus.Err()
	}

	valuePool, errPool := s.calcBuyFromPool(ctx, valueToBuy, cState, coinFrom, coinTo, route, commissionPoolSwapper)
	if errPool != nil {
		return nil, nil, errPool
	}
	return commission, valuePool, nil
}

func reverseCoinIds(a []uint64) {
	if len(a) == 0 {
		return
	}
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}
