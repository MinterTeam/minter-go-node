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

type SellAllCoinData struct {
	CoinToSell        types.CoinSymbol `json:"coin_to_sell"`
	CoinToBuy         types.CoinSymbol `json:"coin_to_buy"`
	MinimumValueToBuy *big.Int         `json:"minimum_value_to_buy"`
}

func (data SellAllCoinData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	sender, _ := tx.Sender()

	total := TotalSpends{}
	var conversions []Conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	available := context.GetBalance(sender, data.CoinToSell)
	var value *big.Int

	total.Add(data.CoinToSell, available)

	switch {
	case data.CoinToSell.IsBaseCoin():
		amountToSell := big.NewInt(0).Set(available)
		amountToSell.Sub(amountToSell, commissionInBaseCoin)

		coin := context.GetStateCoin(data.CoinToBuy).Data()
		value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, amountToSell)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if err := CheckForCoinSupplyOverflow(coin.Volume, value); err != nil {
			return nil, nil, nil, &Response{
				Code: code.CoinSupplyOverflow,
				Log:  err.Error(),
			}
		}

		conversions = append(conversions, Conversion{
			FromCoin:  data.CoinToSell,
			ToCoin:    data.CoinToBuy,
			ToAmount:  value,
			ToReserve: amountToSell,
		})
	case data.CoinToBuy.IsBaseCoin():
		amountToSell := big.NewInt(0).Set(available)

		coin := context.GetStateCoin(data.CoinToSell).Data()
		ret := formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, amountToSell)

		if ret.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), ret.String()),
			}
		}

		if ret.Cmp(commissionInBaseCoin) == -1 {
			return nil, nil, nil, &Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account"),
			}
		}

		value = big.NewInt(0).Set(ret)
		value.Sub(ret, commissionInBaseCoin)

		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  amountToSell,
			FromReserve: ret,
			ToCoin:      data.CoinToBuy,
		})
	default:
		amountToSell := big.NewInt(0).Set(available)

		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()

		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, amountToSell)
		if basecoinValue.Cmp(commissionInBaseCoin) == -1 {
			return nil, nil, nil, &Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account"),
			}
		}

		basecoinValue.Sub(basecoinValue, commissionInBaseCoin)

		value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)
		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return nil, nil, nil, &Response{
				Code: code.MinimumValueToBuyReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if err := CheckForCoinSupplyOverflow(coinTo.Volume, value); err != nil {
			return nil, nil, nil, &Response{
				Code: code.CoinSupplyOverflow,
				Log:  err.Error(),
			}
		}

		conversions = append(conversions, Conversion{
			FromCoin:    data.CoinToSell,
			FromAmount:  amountToSell,
			FromReserve: big.NewInt(0).Add(basecoinValue, commissionInBaseCoin),
			ToCoin:      data.CoinToBuy,
			ToAmount:    value,
			ToReserve:   basecoinValue,
		})
	}

	return total, conversions, value, nil
}

func (data SellAllCoinData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
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

	return nil
}

func (data SellAllCoinData) String() string {
	return fmt.Sprintf("SELL ALL COIN sell:%s buy:%s",
		data.CoinToSell.String(), data.CoinToBuy.String())
}

func (data SellAllCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data SellAllCoinData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	available := context.GetBalance(sender, data.CoinToSell)

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
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSellAllCoin)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		common.KVPair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		common.KVPair{Key: []byte("tx.return"), Value: []byte(value.String())},
		common.KVPair{Key: []byte("tx.sell_amount"), Value: []byte(available.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
