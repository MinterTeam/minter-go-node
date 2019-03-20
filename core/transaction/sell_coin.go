package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
)

type SellCoinData struct {
	CoinToSell        types.CoinSymbol `json:"coin_to_sell"`
	ValueToSell       *big.Int         `json:"value_to_sell"`
	CoinToBuy         types.CoinSymbol `json:"coin_to_buy"`
	MinimumValueToBuy *big.Int         `json:"minimum_value_to_buy"`
}

func (data SellCoinData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	total := TotalSpends{}
	var conversions []Conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionIncluded := false

	var value *big.Int

	switch {
	case data.CoinToSell.IsBaseCoin():
		coin := context.GetStateCoin(data.CoinToBuy).Data()
		value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToSell)
		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coin.Volume)
			nVolume.Add(nVolume, value)

			nReserveBalance := big.NewInt(0).Set(coin.ReserveBalance)
			nReserveBalance.Add(nReserveBalance, data.ValueToSell)

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coin.Crr, commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		if err := CheckForCoinSupplyOverflow(coin.Volume, value); err != nil {
			return nil, nil, nil, &Response{
				Code: code.CoinSupplyOverflow,
				Log:  err.Error(),
			}
		}

		total.Add(data.CoinToSell, data.ValueToSell)
		conversions = append(conversions, Conversion{
			FromCoin:  data.CoinToSell,
			ToCoin:    data.CoinToBuy,
			ToAmount:  value,
			ToReserve: data.ValueToSell,
		})
	case data.CoinToBuy.IsBaseCoin():
		coin := context.GetStateCoin(data.CoinToSell).Data()
		value = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToSell)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		rValue := big.NewInt(0).Set(value)
		valueToSell := data.ValueToSell

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true

			newVolume := big.NewInt(0).Set(coin.Volume)
			newReserve := big.NewInt(0).Set(coin.ReserveBalance)

			newVolume.Sub(newVolume, data.ValueToSell)
			newReserve.Sub(newReserve, value)

			if newReserve.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.ReserveBalance.String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin())}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coin.Crr, commissionInBaseCoin)

			valueToSell.Add(valueToSell, c)
			rValue.Add(rValue, commissionInBaseCoin)
		}

		total.Add(data.CoinToSell, valueToSell)
		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  valueToSell,
			FromReserve: rValue,
			ToCoin:      data.CoinToBuy,
		})
	default:
		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()

		valueToSell := big.NewInt(0).Set(data.ValueToSell)

		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, data.ValueToSell)

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true
			newVolume := big.NewInt(0).Set(coinFrom.Volume)
			newReserve := big.NewInt(0).Set(coinFrom.ReserveBalance)

			newVolume.Sub(newVolume, data.ValueToSell)
			newReserve.Sub(newReserve, basecoinValue)

			if newReserve.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coinFrom.ReserveBalance.String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin())}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coinFrom.Crr, commissionInBaseCoin)

			valueToSell.Add(valueToSell, c)
			basecoinValue.Add(basecoinValue, commissionInBaseCoin)
		}

		value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coinTo.Volume)
			nVolume.Add(nVolume, value)

			nReserveBalance := big.NewInt(0).Set(coinTo.ReserveBalance)
			nReserveBalance.Add(nReserveBalance, basecoinValue)

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinTo.Crr, commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		if err := CheckForCoinSupplyOverflow(coinTo.Volume, value); err != nil {
			return nil, nil, nil, &Response{
				Code: code.CoinSupplyOverflow,
				Log:  err.Error(),
			}
		}

		total.Add(data.CoinToSell, valueToSell)

		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  valueToSell,
			FromReserve: basecoinValue,
			ToCoin:      data.CoinToBuy,
			ToAmount:    value,
			ToReserve:   basecoinValue,
		})
	}

	if !commissionIncluded {
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if !tx.GasCoin.IsBaseCoin() {
			coin := context.GetStateCoin(tx.GasCoin)

			if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.ReserveBalance().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin())}
			}

			commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		total.Add(tx.GasCoin, commission)
	}

	return total, conversions, value, nil
}

func (data SellCoinData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	if data.ValueToSell == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
	}

	if !context.CoinExists(data.CoinToSell) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists")}
	}

	if !context.CoinExists(data.CoinToBuy) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists")}
	}

	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	return nil
}

func (data SellCoinData) String() string {
	return fmt.Sprintf("SELL COIN sell:%s %s buy:%s",
		data.ValueToSell.String(), data.CoinToBuy.String(), data.CoinToSell.String())
}

func (data SellCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data SellCoinData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	totalSpends, conversions, value, response := data.TotalSpend(tx, context)
	if response != nil {
		return *response
	}

	for _, ts := range totalSpends {
		if context.GetBalance(sender, ts.Coin).Cmp(ts.Value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log: fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s.",
					sender.String(),
					ts.Value.String(),
					ts.Coin)}
		}
	}

	if !isCheck {
		for _, ts := range totalSpends {
			context.SubBalance(sender, ts.Coin, ts.Value)
		}

		for _, conversion := range conversions {
			context.SubCoinVolume(conversion.FromCoin, conversion.FromAmount)
			context.SubCoinReserve(conversion.FromCoin, conversion.FromReserve)

			context.AddCoinVolume(conversion.ToCoin, conversion.ToAmount)
			context.AddCoinReserve(conversion.ToCoin, conversion.ToReserve)
		}

		rewardPool.Add(rewardPool, tx.CommissionInBaseCoin())
		context.AddBalance(sender, data.CoinToBuy, value)
		context.SetNonce(sender, tx.Nonce)

		context.DeleteCoinIfZeroReserve(data.CoinToBuy)
		context.DeleteCoinIfZeroReserve(data.CoinToSell)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSellCoin)}))},
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
