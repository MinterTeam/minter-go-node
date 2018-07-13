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
		Coin  types.CoinSymbol `json:"coin,string"`
		To    types.Address    `json:"to"`
		Value string           `json:"value"`
	}{
		Coin:  data.Coin,
		To:    data.To,
		Value: data.Value.String(),
	})
}

func (data SendData) String() string {
	return fmt.Sprintf("SEND to:%s coin:%s value:%s",
		data.To.String(), data.Coin.String(), data.Value.String())
}

func (data SendData) Gas() int64 {
	return commissions.SendTx
}

func (data SendData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	if !context.CoinExists(data.Coin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists")}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if data.Coin != types.GetBaseCoin() {
		coin := context.GetStateCoin(data.Coin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	totalTxCost := big.NewInt(0).Add(data.Value, commission)

	if context.GetBalance(sender, data.Coin).Cmp(totalTxCost) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commissionInBaseCoin)

		if data.Coin != types.GetBaseCoin() {
			context.SubCoinVolume(data.Coin, commission)
			context.SubCoinReserve(data.Coin, commissionInBaseCoin)
		}

		context.SubBalance(sender, data.Coin, totalTxCost)
		context.AddBalance(data.To, data.Coin, data.Value)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeSend}},
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
