package service

import (
	"context"
	"fmt"
	"math/big"

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
)

// EstimateCoinSell return estimate of sell coin transaction.
func (s *Service) EstimateCoinSell(ctx context.Context, req *pb.EstimateCoinSellRequest) (*pb.EstimateCoinSellResponse, error) {
	valueToSell, ok := big.NewInt(0).SetString(req.ValueToSell, 10)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "Value to sell not specified")
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

	switch req.SwapFrom {
	case pb.SwapFrom_bancor:
		commission, valueBancor, err := s.calcSellBancorWithCommission(commissions, cState, requestCoinCommissionID, coinTo, coinFrom, valueToSell)
		if err != nil {
			return nil, err
		}
		value = valueBancor
		resultCommission = commission
	case pb.SwapFrom_pool:
		commission, valuePool, err := s.calcSellPoolWithCommission(ctx, commissions, cState, requestCoinCommissionID, valueToSell, coinFrom, coinTo, req.Route)
		if err != nil {
			return nil, err
		}
		value = valuePool
		resultCommission = commission
	default:
		commissionBancor, valueBancor, errBancor := s.calcSellBancorWithCommission(commissions, cState, requestCoinCommissionID, coinTo, coinFrom, valueToSell)
		commissionPool, valuePool, errPool := s.calcSellPoolWithCommission(ctx, commissions, cState, requestCoinCommissionID, valueToSell, coinFrom, coinTo, req.Route)

		if valueBancor != nil && valuePool != nil {
			if valueBancor.Cmp(valuePool) == -1 {
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

	res := &pb.EstimateCoinSellResponse{
		WillGet:    value.String(),
		Commission: resultCommission.String(),
		SwapFrom:   swapFrom,
	}
	return res, nil
}

func (s *Service) calcSellBancorWithCommission(commissions *commission.Price, cState *state.CheckState, requestCoinCommissionID types.CoinID, coinTo transaction.CalculateCoin, coinFrom transaction.CalculateCoin, valueToSell *big.Int) (*big.Int, *big.Int, error) {
	commissionInBaseCoin := commissions.SellBancor
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

	valueBancor, errBancor := s.calcSellFromBancor(valueToSell, coinTo, coinFrom)
	if errBancor != nil {
		return nil, nil, errBancor
	}
	return commission, valueBancor, nil
}

func (s *Service) calcSellPoolWithCommission(ctx context.Context, commissions *commission.Price, cState *state.CheckState, requestCoinCommissionID types.CoinID, valueToSell *big.Int, coinFrom transaction.CalculateCoin, coinTo transaction.CalculateCoin, route []uint64) (*big.Int, *big.Int, error) {
	commissionInBaseCoin := big.NewInt(0).Add(commissions.SellPoolBase, big.NewInt(0).Mul(commissions.SellPoolDelta, big.NewInt(int64(len(route)))))
	commission, commissionFromPool, err := s.commissionInCoin(cState, requestCoinCommissionID, commissions.Coin, commissionInBaseCoin)
	if err != nil {
		return nil, nil, err
	}

	commissionPoolSwapper := cState.Swap().GetSwapper(requestCoinCommissionID, types.GetBaseCoinID())
	if commissionFromPool && requestCoinCommissionID != types.GetBaseCoinID() {
		commissionPoolSwapper = commissionPoolSwapper.AddLastSwapStepWithOrders(commission, commissionInBaseCoin)
	}
	if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
		return nil, nil, timeoutStatus.Err()
	}

	valuePool, errPool := s.calcSellFromPool(ctx, valueToSell, cState, coinFrom, coinTo, route, commissionPoolSwapper)
	if errPool != nil {
		return nil, nil, errPool
	}
	return commission, valuePool, nil
}

func (s *Service) commissionInCoin(cState *state.CheckState, coinCommissionID types.CoinID, commissionsCoin types.CoinID, commissionInBaseCoin *big.Int) (*big.Int, bool, error) {

	var isSwapFromPool bool
	var commission *big.Int
	switch coinCommissionID {
	case commissionsCoin:
		commission = commissionInBaseCoin
		isSwapFromPool = true
	case types.GetBaseCoinID():
		commission = cState.Swap().GetSwapper(commissionsCoin, types.GetBaseCoinID()).CalculateBuyForSellWithOrders(commissionInBaseCoin)
		if commission == nil {
			return nil, false, s.createError(status.New(codes.FailedPrecondition, "Not possible to pay commission"), transaction.EncodeError(code.NewCommissionCoinNotSufficient("", "")))
		}
	default:
		if !commissionsCoin.IsBaseCoin() {
			commissionInBaseCoin = cState.Swap().GetSwapper(commissionsCoin, types.GetBaseCoinID()).CalculateBuyForSellWithOrders(commissionInBaseCoin)
			if commissionInBaseCoin == nil {
				return nil, false, s.createError(status.New(codes.FailedPrecondition, "Not possible to pay commission"), transaction.EncodeError(code.NewCommissionCoinNotSufficient("", "")))
			}
		}
		commissionPoolSwapper := cState.Swap().GetSwapper(coinCommissionID, types.GetBaseCoinID())

		comm, fromPool, errResp := transaction.CalculateCommission(cState, commissionPoolSwapper, cState.Coins().GetCoin(coinCommissionID), commissionInBaseCoin)
		if errResp != nil {
			return nil, false, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
		commission = comm
		isSwapFromPool = bool(fromPool)
	}
	return commission, isSwapFromPool, nil
}

func (s *Service) calcSellFromPool(ctx context.Context, value *big.Int, cState *state.CheckState, coinFrom transaction.CalculateCoin, coinTo transaction.CalculateCoin, route []uint64, commissionPoolSwapper swap.EditableChecker) (*big.Int, error) {
	sellCoinID := coinFrom.ID()
	sellValue := big.NewInt(0).Set(value)
	coinSell := coinFrom
	dup := make(map[uint32]struct{})
	for _, buyCoinInt := range append(route, uint64(coinTo.ID())) {
		if timeoutStatus := s.checkTimeout(ctx); timeoutStatus != nil {
			return nil, timeoutStatus.Err()
		}
		buyCoinID := types.CoinID(buyCoinInt)
		swapChecker := cState.Swap().GetSwapper(sellCoinID, buyCoinID)

		coinBuy := coinTo
		if buyCoinID != coinBuy.ID() {
			coinBuy = cState.Coins().GetCoin(buyCoinID)
		}
		if !swapChecker.Exists() {
			return nil, s.createError(status.New(codes.NotFound, fmt.Sprintf("swap pool between coins %s and %s not exists", coinSell.GetFullSymbol(), coinBuy.GetFullSymbol())), transaction.EncodeError(code.NewPairNotExists(coinSell.ID().String(), coinBuy.ID().String())))
		}

		if _, ok := dup[swapChecker.GetID()]; ok {
			return nil, s.createError(status.New(codes.InvalidArgument, fmt.Sprintf("Forbidden to repeat the pool in the route, pool duplicate %d", swapChecker.GetID())), transaction.EncodeError(code.NewDuplicatePoolInRouteCode(swapChecker.GetID())))
		}
		dup[swapChecker.GetID()] = struct{}{}

		if swapChecker.GetID() == commissionPoolSwapper.GetID() {
			if sellCoinID == types.GetBaseCoinID() {
				swapChecker = commissionPoolSwapper.Reverse()
			} else {
				swapChecker = commissionPoolSwapper
			}
		}

		buyValue := swapChecker.CalculateBuyForSellWithOrders(sellValue)
		if buyValue == nil || buyValue.Sign() != 1 {
			reserve0, reserve1 := swapChecker.Reserves()
			return nil, s.createError(status.New(codes.OutOfRange, fmt.Sprintf("swap pool has reserves %s %s and %d %s, you wanted sell %s %s", reserve0, coinSell.GetFullSymbol(), reserve1, coinBuy.GetFullSymbol(), sellValue, coinSell.GetFullSymbol())), "")
		}

		if errResp := transaction.CheckSwap(swapChecker, coinSell, coinBuy, sellValue, buyValue, false); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}

		sellValue = buyValue
		coinSell = coinBuy
		sellCoinID = buyCoinID
	}

	return sellValue, nil
}

func (s *Service) calcSellFromBancor(value *big.Int, coinTo transaction.CalculateCoin, coinFrom transaction.CalculateCoin) (*big.Int, error) {
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

	if !coinFrom.ID().IsBaseCoin() {
		var errResp *transaction.Response
		value, errResp = transaction.CalculateSaleReturnAndCheck(coinFrom, value)
		if errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}

	if !coinTo.ID().IsBaseCoin() {
		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), value)
		if errResp := transaction.CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return nil, s.createError(status.New(codes.FailedPrecondition, errResp.Log), errResp.Info)
		}
	}
	return value, nil
}
