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

type EstimateCoinBuyResponse struct {
	WillPay    string `json:"will_pay"`
	Commission string `json:"commission"`
}

func EstimateCoinBuy(coinToSellString string, coinToBuyString string, valueToBuy *big.Int, height int) (*EstimateCoinBuyResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	coinToSell := types.StrToCoinSymbol(coinToSellString)
	coinToBuy := types.StrToCoinSymbol(coinToBuyString)

	var result *big.Int

	if coinToSell == coinToBuy {
		return nil, rpctypes.RPCError{Code: 400, Message: "\"From\" coin equals to \"to\" coin"}
	}

	if !cState.Coins().Exists(coinToSell) {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to sell not exists"}
	}

	if !cState.Coins().Exists(coinToBuy) {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to buy not exists"}
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if coinToSell != types.GetBaseCoin() {
		coin := cState.Coins().GetCoin(coinToSell)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	switch {
	case coinToSell == types.GetBaseCoin():
		coin := cState.Coins().GetCoin(coinToBuy)
		result = formula.CalculatePurchaseAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)
	case coinToBuy == types.GetBaseCoin():
		coin := cState.Coins().GetCoin(coinToSell)

		if coin.Reserve().Cmp(valueToBuy) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), valueToBuy.String())}
		}

		result = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), valueToBuy)
	default:
		coinFrom := cState.Coins().GetCoin(coinToSell)
		coinTo := cState.Coins().GetCoin(coinToBuy)
		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToBuy)

		if coinFrom.Reserve().Cmp(baseCoinNeeded) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), baseCoinNeeded.String())}
		}

		result = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), baseCoinNeeded)
	}

	return &EstimateCoinBuyResponse{
		WillPay:    result.String(),
		Commission: commission.String(),
	}, nil
}
