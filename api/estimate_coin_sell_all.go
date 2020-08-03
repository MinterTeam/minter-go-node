package api

import (
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
)

type EstimateCoinSellAllResponse struct {
	WillGet string `json:"will_get"`
}

func EstimateCoinSellAll(coinIdToSell uint32, coinIdToBuy uint32, valueToSell *big.Int, gasPrice uint64, height int) (*EstimateCoinSellAllResponse,
	error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	if gasPrice < 1 {
		gasPrice = 1
	}

	sellCoinID := types.CoinID(coinIdToSell)
	buyCoinID := types.CoinID(coinIdToBuy)

	var result *big.Int

	if sellCoinID == buyCoinID {
		return nil, rpctypes.RPCError{Code: 400, Message: "\"From\" coin equals to \"to\" coin"}
	}

	if !cState.Coins().Exists(sellCoinID) {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to sell not exists"}
	}

	if !cState.Coins().Exists(buyCoinID) {
		return nil, rpctypes.RPCError{Code: 404, Message: "Coin to buy not exists"}
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	switch {
	case sellCoinID.IsBaseCoin():
		coin := cState.Coins().GetCoin(buyCoinID)

		valueToSell.Sub(valueToSell, commission)
		if valueToSell.Cmp(big.NewInt(0)) != 1 {
			return nil, rpctypes.RPCError{Code: 400, Message: "Not enough coins to pay commission"}
		}

		result = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	case buyCoinID.IsBaseCoin():
		coin := cState.Coins().GetCoin(sellCoinID)
		result = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)

		result.Sub(result, commission)
		if result.Cmp(big.NewInt(0)) != 1 {
			return nil, rpctypes.RPCError{Code: 400, Message: "Not enough coins to pay commission"}
		}
	default:
		coinFrom := cState.Coins().GetCoin(sellCoinID)
		coinTo := cState.Coins().GetCoin(buyCoinID)
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
