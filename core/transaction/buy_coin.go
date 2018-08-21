package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
)

type BuyCoinData struct {
	CoinToBuy  types.CoinSymbol
	ValueToBuy *big.Int
	CoinToSell types.CoinSymbol
}

func (data BuyCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToBuy  types.CoinSymbol `json:"coin_to_buy,string"`
		ValueToBuy string           `json:"value_to_buy"`
		CoinToSell types.CoinSymbol `json:"coin_to_sell,string"`
	}{
		CoinToBuy:  data.CoinToBuy,
		ValueToBuy: data.ValueToBuy.String(),
		CoinToSell: data.CoinToSell,
	})
}

func (data BuyCoinData) String() string {
	return fmt.Sprintf("BUY COIN sell:%s buy:%s %s",
		data.CoinToSell.String(), data.ValueToBuy.String(), data.CoinToBuy.String())
}

func (data BuyCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data BuyCoinData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	if data.CoinToSell == data.CoinToBuy {
		return Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
	}

	if !context.CoinExists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if !context.CoinExists(data.CoinToSell) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.CoinToSell)}
	}

	if !context.CoinExists(data.CoinToBuy) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.CoinToBuy)}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s", coin.ReserveBalance().String(), types.GetBaseCoin(), commissionInBaseCoin.String(), types.GetBaseCoin())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if context.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s.", sender.String(), commission.String(), tx.GasCoin)}
	}

	var value *big.Int

	if data.CoinToSell.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToBuy).Data()

		value = formula.CalculatePurchaseAmount(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToBuy)

		if context.GetBalance(sender, data.CoinToSell).Cmp(value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value.String(), data.CoinToSell)}
		}

		if data.CoinToSell == tx.GasCoin {
			totalTxCost := big.NewInt(0)
			totalTxCost.Add(totalTxCost, value)
			totalTxCost.Add(totalTxCost, commission)

			if context.GetBalance(sender, data.CoinToSell).Cmp(totalTxCost) < 0 {
				return Response{
					Code: code.InsufficientFunds,
					Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), tx.GasCoin)}
			}
		}

		if !isCheck {
			context.SubBalance(sender, data.CoinToSell, value)
			context.AddCoinVolume(data.CoinToBuy, data.ValueToBuy)
			context.AddCoinReserve(data.CoinToBuy, value)
		}
	} else if data.CoinToBuy.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToSell).Data()

		value = formula.CalculateSaleAmount(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToBuy)

		if context.GetBalance(sender, data.CoinToSell).Cmp(value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value.String(), data.CoinToSell)}
		}

		if data.CoinToSell == tx.GasCoin {
			totalTxCost := big.NewInt(0)
			totalTxCost.Add(totalTxCost, value)
			totalTxCost.Add(totalTxCost, commission)

			if context.GetBalance(sender, data.CoinToSell).Cmp(totalTxCost) < 0 {
				return Response{
					Code: code.InsufficientFunds,
					Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), tx.GasCoin)}
			}
		}

		if !isCheck {
			context.SubBalance(sender, data.CoinToSell, value)
			context.SubCoinVolume(data.CoinToSell, value)
			context.SubCoinReserve(data.CoinToSell, data.ValueToBuy)
		}
	} else {
		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()

		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, data.ValueToBuy)
		value = formula.CalculateSaleAmount(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, baseCoinNeeded)

		if context.GetBalance(sender, data.CoinToSell).Cmp(value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value.String(), data.CoinToSell)}
		}

		if data.CoinToSell == tx.GasCoin {
			totalTxCost := big.NewInt(0)
			totalTxCost.Add(totalTxCost, value)
			totalTxCost.Add(totalTxCost, commission)

			if context.GetBalance(sender, data.CoinToSell).Cmp(totalTxCost) < 0 {
				return Response{
					Code: code.InsufficientFunds,
					Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), tx.GasCoin)}
			}
		}

		if !isCheck {
			context.SubBalance(sender, data.CoinToSell, value)

			context.AddCoinVolume(data.CoinToBuy, data.ValueToBuy)
			context.SubCoinVolume(data.CoinToSell, value)

			context.AddCoinReserve(data.CoinToBuy, baseCoinNeeded)
			context.SubCoinReserve(data.CoinToSell, baseCoinNeeded)
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.SubBalance(sender, tx.GasCoin, commission)

		if !tx.GasCoin.IsBaseCoin() {
			context.SubCoinVolume(tx.GasCoin, commission)
			context.SubCoinReserve(tx.GasCoin, commissionInBaseCoin)
		}

		context.AddBalance(sender, data.CoinToBuy, data.ValueToBuy)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeBuyCoin}},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		common.KVPair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		common.KVPair{Key: []byte("tx.return"), Value: []byte(value.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
