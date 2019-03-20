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

type BuyCoinData struct {
	CoinToBuy          types.CoinSymbol `json:"coin_to_buy"`
	ValueToBuy         *big.Int         `json:"value_to_buy"`
	CoinToSell         types.CoinSymbol `json:"coin_to_sell"`
	MaximumValueToSell *big.Int         `json:"maximum_value_to_sell"`
}

func (data BuyCoinData) String() string {
	return fmt.Sprintf("BUY COIN sell:%s buy:%s %s",
		data.CoinToSell.String(), data.ValueToBuy.String(), data.CoinToBuy.String())
}

func (data BuyCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data BuyCoinData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends,
	[]Conversion, *big.Int, *Response) {
	total := TotalSpends{}
	var conversions []Conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionIncluded := false

	if !data.CoinToBuy.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToBuy).Data()

		if err := CheckForCoinSupplyOverflow(coin.Volume, data.ValueToBuy); err != nil {
			return nil, nil, nil, &Response{
				Code: code.CoinSupplyOverflow,
				Log:  err.Error(),
			}
		}
	}

	var value *big.Int

	switch {
	case data.CoinToSell.IsBaseCoin():
		coin := context.GetStateCoin(data.CoinToBuy).Data()
		value = formula.CalculatePurchaseAmount(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToBuy)

		if value.Cmp(data.MaximumValueToSell) == 1 {
			return nil, nil, nil, &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					data.MaximumValueToSell.String(), value.String()),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coin.Volume)
			nVolume.Add(nVolume, data.ValueToBuy)

			nReserveBalance := big.NewInt(0).Set(coin.ReserveBalance)
			nReserveBalance.Add(nReserveBalance, value)

			if nReserveBalance.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.ReserveBalance.String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin())}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coin.Crr, commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, Conversion{
			FromCoin:  data.CoinToSell,
			ToCoin:    data.CoinToBuy,
			ToAmount:  data.ValueToBuy,
			ToReserve: value,
		})
	case data.CoinToBuy.IsBaseCoin():
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true
			valueToBuy.Add(valueToBuy, commissionInBaseCoin)
		}

		coin := context.GetStateCoin(data.CoinToSell).Data()
		value = formula.CalculateSaleAmount(coin.Volume, coin.ReserveBalance, coin.Crr, valueToBuy)

		if value.Cmp(data.MaximumValueToSell) == 1 {
			return nil, nil, nil, &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					data.MaximumValueToSell.String(), value.String()),
			}
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  value,
			FromReserve: valueToBuy,
			ToCoin:      data.CoinToBuy,
		})
	default:
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true
			valueToBuy.Add(valueToBuy, commissionInBaseCoin)
		}

		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()
		baseCoinNeeded := formula.CalculatePurchaseAmount(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, valueToBuy)

		if coinFrom.ReserveBalance.Cmp(baseCoinNeeded) < 0 {
			return nil, nil, nil, &Response{
				Code: code.CoinReserveNotSufficient,
				Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
					coinFrom.ReserveBalance.String(),
					types.GetBaseCoin(),
					baseCoinNeeded.String(),
					types.GetBaseCoin())}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coinTo.Volume)
			nVolume.Add(nVolume, valueToBuy)

			nReserveBalance := big.NewInt(0).Set(coinTo.ReserveBalance)
			nReserveBalance.Add(nReserveBalance, baseCoinNeeded)

			if nReserveBalance.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coinFrom.ReserveBalance.String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin())}
			}

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinTo.Crr, commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		value = formula.CalculateSaleAmount(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, baseCoinNeeded)

		if value.Cmp(data.MaximumValueToSell) == 1 {
			return nil, nil, nil, &Response{
				Code: code.MaximumValueToSellReached,
				Log:  fmt.Sprintf("You wanted to sell maximum %s, but currently you need to spend %s to complete tx", data.MaximumValueToSell.String(), value.String()),
			}
		}

		total.Add(data.CoinToSell, value)
		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  value,
			FromReserve: baseCoinNeeded,
			ToCoin:      data.CoinToBuy,
			ToAmount:    valueToBuy,
			ToReserve:   baseCoinNeeded,
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

func (data BuyCoinData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	if data.ValueToBuy == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
	}

	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if !context.CoinExists(data.CoinToSell) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.CoinToSell)}
	}

	if !context.CoinExists(data.CoinToBuy) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.CoinToBuy)}
	}

	return nil
}

func (data BuyCoinData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
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
		context.AddBalance(sender, data.CoinToBuy, data.ValueToBuy)
		context.SetNonce(sender, tx.Nonce)

		context.DeleteCoinIfZeroReserve(data.CoinToBuy)
		context.DeleteCoinIfZeroReserve(data.CoinToSell)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeBuyCoin)}))},
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
