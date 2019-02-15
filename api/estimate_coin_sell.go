package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
)

type EstimateCoinSellResponse struct {
	WillGet    *big.Int `json:"will_get"`
	Commission *big.Int `json:"commission"`
}

func EstimateCoinSell(
	coinToSellString string, coinToBuyString string, valueToSell *big.Int, height int) (*EstimateCoinSellResponse,
	error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	coinToSell := types.StrToCoinSymbol(coinToSellString)
	coinToBuy := types.StrToCoinSymbol(coinToBuyString)

	var result *big.Int

	if coinToSell == coinToBuy {
		return nil, rpctypes.RPCError{Code: 400, Message: "\"From\" coin equals to \"to\" coin"}
	}

	if !cState.CoinExists(coinToSell) {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to sell not exists"}
	}

	if !cState.CoinExists(coinToBuy) {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to buy not exists"}
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if coinToSell != types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToSell)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		if coin.Volume().Cmp(valueToSell) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin volume is not sufficient for transaction. Has: %s, required %s",
				coin.Volume().String(), valueToSell.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.GetStateCoin(coinToBuy).Data()
		result = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, valueToSell)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.GetStateCoin(coinToSell).Data()
		result = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, valueToSell)
	default:
		coinFrom := cState.GetStateCoin(coinToSell).Data()
		coinTo := cState.GetStateCoin(coinToBuy).Data()
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, valueToSell)
		result = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)
	}

	return &EstimateCoinSellResponse{
		WillGet:    result,
		Commission: commission,
	}, nil
}
