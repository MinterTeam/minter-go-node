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
	panic("implement me")
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

func (data SellAllCoinData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	available := context.GetBalance(sender, data.CoinToSell)

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !data.CoinToSell.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToSell)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if context.GetBalance(sender, data.CoinToSell).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), commission)}
	}

	amountToSell := big.NewInt(0).Set(available)
	amountToSell.Sub(amountToSell, commission)

	var value *big.Int

	if data.CoinToSell.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToBuy).Data()
		value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, amountToSell)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return Response{
				Code: code.MinimumValueToBuylReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if !isCheck {
			context.AddCoinVolume(data.CoinToBuy, value)
			context.AddCoinReserve(data.CoinToBuy, amountToSell)
		}
	} else if data.CoinToBuy.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToSell).Data()
		value = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, amountToSell)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return Response{
				Code: code.MinimumValueToBuylReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if !isCheck {
			context.SubCoinVolume(data.CoinToSell, amountToSell)
			context.SubCoinReserve(data.CoinToSell, value)
		}
	} else {
		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()

		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, amountToSell)
		value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)

		if value.Cmp(data.MinimumValueToBuy) == -1 {
			return Response{
				Code: code.MinimumValueToBuylReached,
				Log:  fmt.Sprintf("You wanted to get minimum %s, but currently you will get %s", data.MinimumValueToBuy.String(), value.String()),
			}
		}

		if !isCheck {
			context.AddCoinVolume(data.CoinToBuy, value)
			context.SubCoinVolume(data.CoinToSell, amountToSell)

			context.AddCoinReserve(data.CoinToBuy, basecoinValue)
			context.SubCoinReserve(data.CoinToSell, basecoinValue)
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.SubBalance(sender, data.CoinToSell, available)

		if !data.CoinToSell.IsBaseCoin() {
			context.SubCoinVolume(data.CoinToSell, commission)
			context.SubCoinReserve(data.CoinToSell, commissionInBaseCoin)
		}

		context.AddBalance(sender, data.CoinToBuy, value)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{byte(TypeSellAllCoin)}},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		common.KVPair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		common.KVPair{Key: []byte("tx.return"), Value: []byte(value.String())},
		common.KVPair{Key: []byte("tx.sell_amount"), Value: []byte(amountToSell.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
