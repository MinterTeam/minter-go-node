package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/pkg/errors"
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

	coinToSell := types.StrToCoinSymbol(coinToSellString)
	coinToBuy := types.StrToCoinSymbol(coinToBuyString)

	var result *big.Int

	if coinToSell == coinToBuy {
		return nil, errors.New("\"From\" coin equals to \"to\" coin")
	}

	if !cState.CoinExists(coinToSell) {
		return nil, errors.New("Coin to sell not exists")
	}

	if !cState.CoinExists(coinToBuy) {
		return nil, errors.New("Coin to buy not exists")
	}

	commissionInBaseCoin := big.NewInt(commissions.ConvertTx)
	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if coinToSell != types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToSell)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return nil, errors.New(fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String()))
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if coinToSell == types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToBuy).Data()
		result = formula.CalculatePurchaseAmount(coin.Volume, coin.ReserveBalance, coin.Crr, valueToBuy)
	} else if coinToBuy == types.GetBaseCoin() {
		coin := cState.GetStateCoin(coinToSell).Data()
		result = formula.CalculateSaleAmount(coin.Volume, coin.ReserveBalance, coin.Crr, valueToBuy)
	} else {
		coinFrom := cState.GetStateCoin(coinToSell).Data()
		coinTo := cState.GetStateCoin(coinToBuy).Data()
		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, valueToBuy)

		if coinFrom.ReserveBalance.Cmp(baseCoinNeeded) < 0 {
			return nil, errors.New(fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coinFrom.ReserveBalance.String(), baseCoinNeeded.String()))
		}

		result = formula.CalculateSaleAmount(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, baseCoinNeeded)
	}

	return &EstimateCoinBuyResponse{
		WillPay:    result.String(),
		Commission: commission.String(),
	}, nil
}
