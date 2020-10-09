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

// EstimateCoinBuyResponse returns an estimate of buy coin transaction
type EstimateCoinBuyResponse struct {
	WillPay    string `json:"will_pay"`
	Commission string `json:"commission"`
}

// EstimateCoinBuy returns an estimate of buy coin transaction
func EstimateCoinBuy(coinToSell, coinToBuy string, valueToBuy *big.Int, height int) (*EstimateCoinBuyResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	coinFrom := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(coinToSell), types.GetVersionFromSymbol(coinToSell))
	if coinFrom == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to sell not exists"}
	}

	coinTo := cState.Coins().GetCoinBySymbol(types.StrToCoinBaseSymbol(coinToBuy), types.GetVersionFromSymbol(coinToBuy))
	if coinTo == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to buy not exists"}
	}

	if coinFrom.ID() == coinTo.ID() {
		return nil, rpctypes.RPCError{Code: 400, Message: "\"From\" coin equals to \"to\" coin"}
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !coinFrom.ID().IsBaseCoin() {
		if coinFrom.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), commissionInBaseCoin.String())}
		}
		commission = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), commissionInBaseCoin)
	}

	var result *big.Int

	switch {
	case coinTo.ID().IsBaseCoin():
		if coinFrom.Reserve().Cmp(valueToBuy) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), valueToBuy.String())}
		}
		result = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToBuy)
	case coinFrom.ID().IsBaseCoin():
		result = formula.CalculatePurchaseAmount(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToBuy)
	default:
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
