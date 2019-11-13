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

type SendData struct {
	Coin  types.CoinSymbol
	To    types.Address
	Value *big.Int
}

func (data SendData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Coin  string `json:"coin"`
		To    string `json:"to"`
		Value string `json:"value"`
	}{
		Coin:  data.Coin.String(),
		To:    data.To.String(),
		Value: data.Value.String(),
	})
}

func (data SendData) TotalSpend(tx *Transaction, context *state.State) (TotalSpends, []Conversion, *big.Int, *Response) {
	total := TotalSpends{}
	var conversions []Conversion

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.Coins.GetCoin(tx.GasCoin)

		err := coin.CheckReserveUnderflow(commissionInBaseCoin)
		if err != nil {
			return nil, nil, nil, &Response{
				Code: code.CoinReserveUnderflow,
				Log:  err.Error()}
		}

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return nil, nil, nil, &Response{
				Code: code.CoinReserveNotSufficient,
				Log: fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s",
					coin.Reserve().String(),
					commissionInBaseCoin.String())}
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
	total.Add(data.Coin, data.Value)

	return total, conversions, nil, nil
}

func (data SendData) BasicCheck(tx *Transaction, context *state.State) *Response {
	if data.Value == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if !context.Coins.Exists(data.Coin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Coin)}
	}

	return nil
}

func (data SendData) String() string {
	return fmt.Sprintf("SEND to:%s coin:%s value:%s",
		data.To.String(), data.Coin.String(), data.Value.String())
}

func (data SendData) Gas() int64 {
	return commissions.SendTx
}

func (data SendData) Run(tx *Transaction, context *state.State, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	totalSpends, conversions, _, response := data.TotalSpend(tx, context)
	if response != nil {
		return *response
	}

	for _, ts := range totalSpends {
		if context.Accounts.GetBalance(sender, ts.Coin).Cmp(ts.Value) < 0 {
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
			context.Accounts.SubBalance(sender, ts.Coin, ts.Value)
		}

		for _, conversion := range conversions {
			context.Coins.SubVolume(conversion.FromCoin, conversion.FromAmount)
			context.Coins.SubReserve(conversion.FromCoin, conversion.FromReserve)

			context.Coins.AddVolume(conversion.ToCoin, conversion.ToAmount)
			context.Coins.AddReserve(conversion.ToCoin, conversion.ToReserve)
		}

		rewardPool.Add(rewardPool, tx.CommissionInBaseCoin())
		context.Accounts.AddBalance(data.To, data.Coin, data.Value)
		context.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSend)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(data.To[:]))},
		common.KVPair{Key: []byte("tx.coin"), Value: []byte(data.Coin.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
