package api

import (
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
)

// EstimateCoinSellAllResponse returns an of sell all coin transaction
type EstimateCoinSellAllResponse struct {
	WillGet string `json:"will_get"`
}

// EstimateCoinSellAll returns an estimate of sell all coin transaction
func EstimateCoinSellAll(coinToSell, coinToBuy string, valueToSell *big.Int, height int) (*EstimateCoinSellAllResponse,
	error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

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

	var result *big.Int

	switch {
	case coinFrom.ID().IsBaseCoin():
		valueToSell.Sub(valueToSell, commission)
		if valueToSell.Sign() != 1 {
			return nil, rpctypes.RPCError{Code: 400, Message: "Not enough coins to pay commission"}
		}

		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), valueToSell)
	case coinTo.ID().IsBaseCoin():
		result = formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)

		result.Sub(result, commission)
		if result.Cmp(big.NewInt(0)) != 1 {
			return nil, rpctypes.RPCError{Code: 400, Message: "Not enough coins to pay commission"}
		}
	default:
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		basecoinValue.Sub(basecoinValue, commission)
		if basecoinValue.Cmp(big.NewInt(0)) != 1 {
			return nil, rpctypes.RPCError{Code: 400, Message: "Not enough coins to pay commission"}
		}

		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
	}

	return &EstimateCoinSellAllResponse{
		WillGet: result.String(),
	}, nil
}
