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

type SellAllCoinData struct {
	CoinToSell types.CoinSymbol
	CoinToBuy  types.CoinSymbol
}

func (data SellAllCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToSell types.CoinSymbol `json:"coin_to_sell,string"`
		CoinToBuy  types.CoinSymbol `json:"coin_to_buy,string"`
	}{
		CoinToSell: data.CoinToSell,
		CoinToBuy:  data.CoinToBuy,
	})
}

func (data SellAllCoinData) String() string {
	return fmt.Sprintf("SELL ALL COIN sell:%s buy:%s",
		data.CoinToSell.String(), data.CoinToBuy.String())
}

func (data SellAllCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data SellAllCoinData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response {
	if data.CoinToSell == data.CoinToBuy {
		return Response{
			Code: code.CrossConvert,
			Log:  fmt.Sprintf("\"From\" coin equals to \"to\" coin")}
	}

	if !context.CoinExists(data.CoinToSell) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists")}
	}

	if !context.CoinExists(data.CoinToBuy) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists")}
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

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.SubBalance(sender, data.CoinToSell, available)

		if !data.CoinToSell.IsBaseCoin() {
			context.SubCoinVolume(data.CoinToSell, commission)
			context.SubCoinReserve(data.CoinToSell, commissionInBaseCoin)
		}
	}

	var value *big.Int

	if data.CoinToSell.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToBuy).Data()

		value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, amountToSell)

		if !isCheck {
			context.AddCoinVolume(data.CoinToBuy, value)
			context.AddCoinReserve(data.CoinToBuy, amountToSell)
		}
	} else if data.CoinToBuy.IsBaseCoin() {
		coin := context.GetStateCoin(data.CoinToSell).Data()

		value = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, amountToSell)

		if !isCheck {
			context.SubCoinVolume(data.CoinToSell, amountToSell)
			context.SubCoinReserve(data.CoinToSell, value)
		}
	} else {
		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()

		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, amountToSell)
		value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)

		if !isCheck {
			context.AddCoinVolume(data.CoinToBuy, value)
			context.SubCoinVolume(data.CoinToSell, amountToSell)

			context.AddCoinReserve(data.CoinToBuy, basecoinValue)
			context.SubCoinReserve(data.CoinToSell, basecoinValue)
		}
	}

	if !isCheck {
		context.AddBalance(sender, data.CoinToBuy, value)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeSellCoin}},
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
