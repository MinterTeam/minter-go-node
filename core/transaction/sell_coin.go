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

type SellCoinData struct {
	CoinToSell  types.CoinSymbol
	ValueToSell *big.Int
	CoinToBuy   types.CoinSymbol
}

func (data SellCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToSell  types.CoinSymbol `json:"coin_to_sell,string"`
		ValueToSell string           `json:"value_to_sell"`
		CoinToBuy   types.CoinSymbol `json:"coin_to_buy,string"`
	}{
		CoinToSell:  data.CoinToSell,
		ValueToSell: data.ValueToSell.String(),
		CoinToBuy:   data.CoinToBuy,
	})
}

func (data SellCoinData) String() string {
	return fmt.Sprintf("SELL COIN sell:%s %s buy:%s",
		data.ValueToSell.String(), data.CoinToBuy.String(), data.CoinToSell.String())
}

func (data SellCoinData) Gas() int64 {
	return commissions.ConvertTx
}

func (data SellCoinData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
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

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if data.CoinToSell != types.GetBaseCoin() {
		coin := context.GetStateCoin(data.CoinToSell)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	totalTxCost := big.NewInt(0).Add(data.ValueToSell, commission)

	if context.GetBalance(sender, data.CoinToSell).Cmp(totalTxCost) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commissionInBaseCoin)

		context.SubBalance(sender, data.CoinToSell, totalTxCost)

		if data.CoinToSell != types.GetBaseCoin() {
			context.SubCoinVolume(data.CoinToSell, commission)
			context.SubCoinReserve(data.CoinToSell, commissionInBaseCoin)
		}
	}

	var value *big.Int

	if data.CoinToSell == types.GetBaseCoin() {
		coin := context.GetStateCoin(data.CoinToBuy).Data()

		value = formula.CalculatePurchaseReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToSell)

		if !isCheck {
			context.AddCoinVolume(data.CoinToBuy, value)
			context.AddCoinReserve(data.CoinToBuy, data.ValueToSell)
		}
	} else if data.CoinToBuy == types.GetBaseCoin() {
		coin := context.GetStateCoin(data.CoinToSell).Data()

		value = formula.CalculateSaleReturn(coin.Volume, coin.ReserveBalance, coin.Crr, data.ValueToSell)

		if !isCheck {
			context.SubCoinVolume(data.CoinToSell, data.ValueToSell)
			context.SubCoinReserve(data.CoinToSell, value)
		}
	} else {
		coinFrom := context.GetStateCoin(data.CoinToSell).Data()
		coinTo := context.GetStateCoin(data.CoinToBuy).Data()

		basecoinValue := formula.CalculateSaleReturn(coinFrom.Volume, coinFrom.ReserveBalance, coinFrom.Crr, data.ValueToSell)
		value = formula.CalculatePurchaseReturn(coinTo.Volume, coinTo.ReserveBalance, coinTo.Crr, basecoinValue)

		if !isCheck {
			context.AddCoinVolume(data.CoinToBuy, value)
			context.SubCoinVolume(data.CoinToSell, data.ValueToSell)

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
		common.KVPair{Key: []byte("tx.return"), Value: value.Bytes()},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
