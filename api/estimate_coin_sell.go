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
	WillGet    string `json:"will_get"`
	Commission string `json:"commission"`
}

func EstimateCoinSell(coinIdToSell uint64, coinIdToBuy uint64, valueToSell *big.Int, height int) (*EstimateCoinSellResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

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

	if !sellCoinID.IsBaseCoin() {
		coin := cState.Coins().GetCoin(sellCoinID)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
				coin.Reserve().String(), commissionInBaseCoin.String())}
		}

		if coin.Volume().Cmp(valueToSell) < 0 {
			return nil, rpctypes.RPCError{Code: 400, Message: fmt.Sprintf("Coin volume is not sufficient for transaction. Has: %s, required %s",
				coin.Volume().String(), valueToSell.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	switch {
	case sellCoinID.IsBaseCoin():
		coin := cState.Coins().GetCoin(buyCoinID)
		result = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	case buyCoinID.IsBaseCoin():
		coin := cState.Coins().GetCoin(sellCoinID)
		result = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), valueToSell)
	default:
		coinFrom := cState.Coins().GetCoin(sellCoinID)
		coinTo := cState.Coins().GetCoin(buyCoinID)
		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), valueToSell)
		result = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)
	}

	return &EstimateCoinSellResponse{
		WillGet:    result.String(),
		Commission: commission.String(),
	}, nil
}
