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
)

type SellCoinData struct {
	CoinToSell        types.CoinID
	ValueToSell       *big.Int
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
}

func (data SellCoinData) totalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []conversion, *big.Int, *Response) {
	total := TotalSpends{}
	var conversions []conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionIncluded := false

	var value *big.Int

	coinToBuy := context.Coins().GetCoin(data.CoinToBuy)
	coinToSell := context.Coins().GetCoin(data.CoinToSell)

	switch {
	case data.CoinToSell.IsBaseCoin():
		value = formula.CalculatePurchaseReturn(coinToBuy.Volume(), coinToBuy.Reserve(), coinToBuy.Crr(), data.ValueToSell)
		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
				Info: EncodeError(code.NewMinimumValueToBuyReached(data.MinimumValueToBuy.String(), value.String(), coinToBuy.GetFullSymbol(), coinToBuy.ID().String())),
			}
		}

		if tx.GasCoin == data.CoinToBuy {
			commissionIncluded = true

			nVolume := big.NewInt(0).Set(coinToBuy.Volume())
			nVolume.Add(nVolume, value)

			nReserveBalance := big.NewInt(0).Set(coinToBuy.Reserve())
			nReserveBalance.Add(nReserveBalance, data.ValueToSell)

			commission := formula.CalculateSaleAmount(nVolume, nReserveBalance, coinToBuy.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, commission)
			conversions = append(conversions, conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		if errResp := CheckForCoinSupplyOverflow(coinToBuy, value); errResp != nil {
			return nil, nil, nil, errResp
		}

		total.Add(data.CoinToSell, data.ValueToSell)
		conversions = append(conversions, conversion{
			FromCoin:  data.CoinToSell,
			ToCoin:    data.CoinToBuy,
			ToAmount:  value,
			ToReserve: data.ValueToSell,
		})
	case data.CoinToBuy.IsBaseCoin():
		value = formula.CalculateSaleReturn(coinToSell.Volume(), coinToSell.Reserve(), coinToSell.Crr(), data.ValueToSell)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
				Info: EncodeError(code.NewMinimumValueToBuyReached(data.MinimumValueToBuy.String(), value.String(), coinToSell.GetFullSymbol(), coinToSell.ID().String())),
			}
		}

		rValue := big.NewInt(0).Set(value)
		valueToSell := data.ValueToSell

		if tx.GasCoin == data.CoinToSell {
			commissionIncluded = true

			newVolume := big.NewInt(0).Set(coinToSell.Volume())
			newReserve := big.NewInt(0).Set(coinToSell.Reserve())

			newVolume.Sub(newVolume, data.ValueToSell)
			newReserve.Sub(newReserve, value)

			if newReserve.Cmp(commissionInBaseCoin) == -1 {
				return nil, nil, nil, &Response{
					Code: code.CoinReserveNotSufficient,
					Log: fmt.Sprintf("Gas coin reserve balance is not sufficient for transaction. Has: %s %s, required %s %s",
						coinToSell.Reserve().String(),
						types.GetBaseCoin(),
						commissionInBaseCoin.String(),
						types.GetBaseCoin()),
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coinToSell.GetFullSymbol(),
						coinToSell.ID().String(),
						coinToSell.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coinToSell.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, c)
			conversions = append(conversions, conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  c,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		total.Add(data.CoinToSell, valueToSell)
		conversions = append(conversions, conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  valueToSell,
			FromReserve: rValue,
			ToCoin:      data.CoinToBuy,
		})
	default:
		coinFrom := coinToSell
		coinTo := coinToBuy

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
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coinFrom.GetFullSymbol(),
						coinFrom.ID().String(),
						coinFrom.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			c := formula.CalculateSaleAmount(newVolume, newReserve, coinFrom.Crr(), commissionInBaseCoin)

			total.Add(tx.GasCoin, c)
			conversions = append(conversions, conversion{
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
				Info: EncodeError(code.NewMinimumValueToBuyReached(data.MinimumValueToBuy.String(), value.String(), coinTo.GetFullSymbol(), coinTo.ID().String())),
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
			conversions = append(conversions, conversion{
				FromCoin:    tx.GasCoin,
				FromAmount:  commission,
				FromReserve: commissionInBaseCoin,
				ToCoin:      types.GetBaseCoinID(),
			})
		}

		if errResp := CheckForCoinSupplyOverflow(coinTo, value); errResp != nil {
			return nil, nil, nil, errResp
		}

		total.Add(data.CoinToSell, valueToSell)

		conversions = append(conversions, conversion{
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
					Info: EncodeError(code.NewCoinReserveNotSufficient(
						coin.GetFullSymbol(),
						coin.ID().String(),
						coin.Reserve().String(),
						commissionInBaseCoin.String(),
					)),
				}
			}

			commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
			conversions = append(conversions, conversion{
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
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	coinToSell := context.Coins().GetCoin(data.CoinToSell)
	if coinToSell == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin to sell not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.CoinToSell.String())),
		}
	}

	if coinToSell.Reserve() == nil {
		return &Response{
			Code: code.CoinReserveNotSufficient, // todo
			Log:  "todo",                        // todo
			Info: EncodeError(code.NewCoinReserveNotSufficient(
				coinToSell.GetFullSymbol(),
				coinToSell.ID().String(),
				coinToSell.Reserve().String(),
				"todo", // todo
			)),
		}
	}

	coinToBuy := context.Coins().GetCoin(data.CoinToBuy)
	if coinToBuy == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin to buy not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.CoinToBuy.String())),
		}
	}

	if coinToBuy.Reserve() == nil {
		return &Response{
			Code: code.CoinReserveNotSufficient, // todo
			Log:  "todo",                        // todo
			Info: EncodeError(code.NewCoinReserveNotSufficient(
				coinToBuy.GetFullSymbol(),
				coinToBuy.ID().String(),
				coinToBuy.Reserve().String(),
				"todo", // todo
			)),
		}
	}

	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToSell.String(),
				coinToSell.GetFullSymbol(),
				data.CoinToBuy.String(),
				coinToBuy.GetFullSymbol()),
			),
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
		checkState = state.NewCheckState(context.(*state.State), nil)
	}

	response := data.BasicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	totalSpends, conversions, value, response := data.totalSpend(tx, checkState)
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
					coin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), ts.Value.String(), coin.GetFullSymbol(), coin.ID().String())),
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
