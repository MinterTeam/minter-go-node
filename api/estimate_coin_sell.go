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

// EstimateCoinSellResponse is estimate of sell coin transaction
type EstimateCoinSellResponse struct {
	WillGet    string `json:"will_get"`
	Commission string `json:"commission"`
}

// EstimateCoinSell returns estimate of sell coin transaction
func EstimateCoinSell(coinToSell, coinToBuy string, valueToSell *big.Int, height int) (*EstimateCoinSellResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	coinFrom := cState.Coins().GetCoinBySymbol(types.StrToCoinSymbol(coinToSell), 0)
	if coinFrom == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to sell not exists"}
	}

	coinTo := cState.Coins().GetCoinBySymbol(types.StrToCoinSymbol(coinToBuy), 0)
	if coinTo == nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to buy not exists"}
	}

	if coinFrom.ID() == coinTo.ID() {
		return nil, rpctypes.RPCError{Code: 400, Message: "\"From\" coin equals to \"to\" coin"}
	}

	var result *big.Int

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !coinFrom.ID().IsBaseCoin() {
		if coinFrom.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Reserve().String(), commissionInBaseCoin.String())}
		}

		if coinFrom.Volume().Cmp(valueToSell) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin volume is not sufficient for transaction. Has: %s, required %s",
				coinFrom.Volume().String(), valueToSell.String())}
		}

		commission = formula.CalculateSaleAmount(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), commissionInBaseCoin)
	}

	switch {
	case coinFrom.ID().IsBaseCoin():
		result = formula.CalculatePurchaseReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
	case coinTo.ID().IsBaseCoin():
		result = formula.CalculateSaleReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToSell)
	default:
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
	}

	return &EstimateCoinSellResponse{
		WillGet:    result.String(),
		Commission: commission.String(),
	}, nil
}
