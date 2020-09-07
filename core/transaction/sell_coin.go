package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
	"strconv"
)

type SellCoinData struct {
	CoinToSell        types.CoinID
	ValueToSell       *big.Int
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
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
				Info: EncodeError(map[string]string{
					"code": strconv.Itoa(int(code.MinimumValueToBuyReached)),
				}),
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
				ToCoin:      types.GetBaseCoinID(),
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
					"code":                 strconv.Itoa(int(code.MinimumValueToBuyReached)),
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
						"code":           strconv.Itoa(int(code.CoinReserveNotSufficient)),
						"has_value":      coin.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"coin_symbol":    fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coin.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, c)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  c,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
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
						"code":           strconv.Itoa(int(code.CoinReserveNotSufficient)),
						"has_value":      coinFrom.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"coin_symbol":    fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coinFrom.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, c)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  c,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		value = formula.CalculatePurchaseReturn(coinTo.Volume(), coinTo.Reserve(), coinTo.Crr(), basecoinValue)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
				Info: EncodeError(map[string]string{
					"code":                 strconv.Itoa(int(code.MinimumValueToBuyReached)),
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
				ToCoin:      types.GetBaseCoinID(),
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
						"code":           strconv.Itoa(int(code.CoinReserveNotSufficient)),
						"has_value":      coin.Reserve().String(),
						"required_value": commissionInBaseCoin.String(),
						"coin_symbol":    fmt.Sprintf("%s", types.GetBaseCoin()),
					}),
				}
			}

			commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
			conversions = append(conversions, Conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
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
			Log:  "Incorrect tx data",
			Info: EncodeError(map[string]string{
				"code": strconv.Itoa(int(code.CrossConvert)),
			}),
		}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin"),
			Info: EncodeError(map[string]string{
				"code":         strconv.Itoa(int(code.CrossConvert)),
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
				"code":    strconv.Itoa(int(code.CoinNotExists)),
				"coin_id": fmt.Sprintf("%s", data.CoinToSell.String()),
			}),
		}
	}

	if !context.Coins().Exists(data.CoinToBuy) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists"),
			Info: EncodeError(map[string]string{
				"code":    strconv.Itoa(int(code.CoinNotExists)),
				"coin_id": fmt.Sprintf("%s", data.CoinToBuy.String()),
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
			coin := checkState.Coins().GetCoin(ts.Coin)

			return Response{
				Code: code.InsufficientFunds,
				Log: fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s.",
					sender.String(),
					ts.Value.String(),
					coin.GetFullSymbol(),
				),
				Info: EncodeError(map[string]string{
					"code":         strconv.Itoa(int(code.InsufficientFunds)),
					"sender":       sender.String(),
					"needed_value": ts.Value.String(),
					"coin_symbol":  coin.GetFullSymbol(),
				}),
			}
		}
	}

	errResp := checkConversionsReserveUnderflow(conversions, checkState)
	if errResp != nil {
		return *errResp
	}

	if deliverState, ok := context.(*state.State); ok {
		for _, ts := range totalSpends {
			deliverState.Accounts.SubBalance(sender, ts.Coin, ts.Value)
		}

		for _, conversion := range conversions {
			deliverState.Coins.SubVolume(conversion.FromCoin, conversion.FromAmount)
			deliverState.Coins.SubReserve(conversion.FromCoin, conversion.FromReserve)

			deliverState.Coins.AddVolume(conversion.ToCoin, conversion.ToAmount)
			deliverState.Coins.AddReserve(conversion.ToCoin, conversion.ToReserve)
		}

		rewardPool.Add(rewardPool, tx.CommissionInBaseCoin())
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, value)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
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
