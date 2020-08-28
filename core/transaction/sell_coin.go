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
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type SellCoinData struct {
	CoinToSell        types.CoinSymbol
	ValueToSell       *big.Int
	CoinToBuy         types.CoinSymbol
	MinimumValueToBuy *big.Int
}

func (data SellCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToSell        string `json:"coin_to_sell"`
		ValueToSell       string `json:"value_to_sell"`
		CoinToBuy         string `json:"coin_to_buy"`
		MinimumValueToBuy string `json:"minimum_value_to_buy"`
	}{
		CoinToSell:        data.CoinToSell.String(),
		ValueToSell:       data.ValueToSell.String(),
		CoinToBuy:         data.CoinToBuy.String(),
		MinimumValueToBuy: data.MinimumValueToBuy.String(),
	})
}

func (data SellCoinData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []Conversion, *big.Int, *Response) {
	total := TotalSpends{}
	var conversions []Conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionIncluded := false

	var value *big.Int

	switch {
	case data.CoinToSell.IsBaseCoin():
		coin := context.Coins().GetCoin(data.CoinToBuy)
		value = formula.CalculatePurchaseReturn(coin.Volume(), coin.Reserve(), coin.Crr(), data.ValueToSell)
		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coin.Volume())
			nVolume.Add(nVolume, value)

			nReserveBalance := big.NewInt(0).Set(coin.Reserve())
			nReserveBalance.Add(nReserveBalance, data.ValueToSell)

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coin.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		if errResp := CheckForCoinSupplyOverflow(coin.Volume(), value, coin.MaxSupply()); errResp != nil {
			return nil, nil, nil, errResp
		}

		total.Add(data.CoinToSell, data.ValueToSell)
		conversions = append(conversions, Conversion{
			FromCoin:  data.CoinToSell,
			ToCoin:    data.CoinToBuy,
			ToAmount:  value,
			ToReserve: data.ValueToSell,
		})
	case data.CoinToBuy.IsBaseCoin():
		coin := context.Coins().GetCoin(data.CoinToSell)
		value = formula.CalculateSaleReturn(coin.Volume(), coin.Reserve(), coin.Crr(), data.ValueToSell)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
				Info: EncodeError(map[string]string{
					"minimum_value_to_buy": data.MinimumValueToBuy.String(),
					"value_to_buy":         value.String(),
				}),
			}
		}

		rValue := big.NewInt(0).Set(value)
		valueToSell := data.ValueToSell

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true

			newVolume := big.NewInt(0).Set(coin.Volume())
			newReserve := big.NewInt(0).Set(coin.Reserve())

			newVolume.Sub(newVolume, data.ValueToSell)
			newReserve.Sub(newReserve, value)

			if newReserve.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has_value":      coin.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coin.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, c)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  c,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		total.Add(data.CoinToSell, valueToSell)
		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  valueToSell,
			FromReserve: rValue,
			ToCoin:      data.CoinToBuy,
		})
	default:
		coinFrom := context.Coins().GetCoin(data.CoinToSell)
		coinTo := context.Coins().GetCoin(data.CoinToBuy)

		valueToSell := big.NewInt(0).Set(data.ValueToSell)

		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume(), coinFrom.Reserve(), coinFrom.Crr(), data.ValueToSell)
		fromReserve := big.NewInt(0).Set(basecoinValue)

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true
			newVolume := big.NewInt(0).Set(coinFrom.Volume())
			newReserve := big.NewInt(0).Set(coinFrom.Reserve())

			newVolume.Sub(newVolume, data.ValueToSell)
			newReserve.Sub(newReserve, basecoinValue)

			if newReserve.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coinFrom.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has":      coinFrom.Reserve().String(),
						"required": commissionInBaseCoin.String(),
						"gas_coin": fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coinFrom.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, c)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  c,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
				Info: EncodeError(map[string]string{
					"minimum_value_to_buy": data.MinimumValueToBuy.String(),
					"get_value":            value.String(),
				}),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coinTo.Volume())
			nVolume.Add(nVolume, value)

			nReserveBalance := big.NewInt(0).Set(coinTo.Reserve())
			nReserveBalance.Add(nReserveBalance, basecoinValue)

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinTo.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoin(),
			})
		}

		if errResp := CheckForCoinSupplyOverflow(coinTo.Volume(), value, coinTo.MaxSupply()); errResp != nil {
			return nil, nil, nil, errResp
		}

		total.Add(data.CoinToSell, valueToSell)

		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  valueToSell,
			FromReserve: fromReserve,
			ToCoin:      data.CoinToBuy,
			ToAmount:    value,
			ToReserve:   basecoinValue,
		})
	}

	if !commissionIncluded {
		commission := big.NewInt(0).Set(commissionInBaseCoin)

		if !tx.GasCoin.IsBaseCoin() {
			coin := context.Coins().GetCoin(tx.GasCoin)

			if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coin.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(map[string]string{
						"has_value":      coin.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"gas_coin":       fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
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

func (data SellCoinData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.ValueToSell == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin"),
			Info: EncodeError(map[string]string{
				"coin_to_sell": fmt.Sprintf("%s", data.CoinToSell),
				"coin_to_buy":  fmt.Sprintf("%s", data.CoinToBuy),
			}),
		}
	}

	if !context.Coins().Exists(data.CoinToSell) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists"),
			Info: EncodeError(map[string]string{
				"coin_to_sell": fmt.Sprintf("%s", data.CoinToSell),
			}),
		}
	}

	if !context.Coins().Exists(data.CoinToBuy) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists"),
			Info: EncodeError(map[string]string{
				"coin_to_buy": fmt.Sprintf("%s", data.CoinToBuy),
			}),
		}
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

func (data SellCoinData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()
	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.BasicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	totalSpends, conversions, value, response := data.TotalSpend(tx, checkState)
	if response != nil {
		return *response
	}

	for _, ts := range totalSpends {
		if checkState.Accounts().GetBalance(sender, ts.Coin).Cmp(ts.Value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log: fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s.",
					sender.String(),
					ts.Value.String(),
					ts.Coin),
				Info: EncodeError(map[string]string{
					"sender":       sender.String(),
					"needed_value": ts.Value.String(),
					"coin":         fmt.Sprintf("%s", ts.Coin),
				}),
			}
		}
	}

	errResp := checkConversionsReserveUnderflow(conversions, checkState)
	if errResp != nil {
		return *errResp
	}

	if deliveryState, ok := context.(*state.State); ok {
		for _, ts := range totalSpends {
			deliveryState.Accounts.SubBalance(sender, ts.Coin, ts.Value)
		}

		for _, conversion := range conversions {
			deliveryState.Coins.SubVolume(conversion.FromCoin, conversion.FromAmount)
			deliveryState.Coins.SubReserve(conversion.FromCoin, conversion.FromReserve)

			deliveryState.Coins.AddVolume(conversion.ToCoin, conversion.ToAmount)
			deliveryState.Coins.AddReserve(conversion.ToCoin, conversion.ToReserve)
		}

		rewardPool.Add(rewardPool, tx.CommissionInBaseCoin())
		deliveryState.Accounts.AddBalance(sender, data.CoinToBuy, value)
		deliveryState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSellCoin)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		kv.Pair{Key: []byte("tx.return"), Value: []byte(value.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
