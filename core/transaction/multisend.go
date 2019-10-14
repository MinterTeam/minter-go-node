package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
	"sort"
	"strings"
)

type MultisendData struct {
	List []MultisendDataItem `json:"list"`
}

func (data MultisendData) TotalSpend(tx *Transaction, context *state.State) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data MultisendData) BasicCheck(tx *Transaction, context *state.State) *Response {
	if len(data.List) < 1 || len(data.List) > 100 {
		return &Response{
			Code: code.InvalidMultisendData,
			Log:  "List length must be between 1 and 100"}
	}

	if err := checkCoins(context, data.List); err != nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  err.Error()}
	}

	return nil
}

type MultisendDataItem struct {
	Coin  types.CoinSymbol `json:"coin"`
	To    types.Address    `json:"to"`
	Value *big.Int         `json:"value"`
}

func (data MultisendData) String() string {
	return fmt.Sprintf("MULTISEND")
}

func (data MultisendData) Gas() int64 {
	return commissions.SendTx + ((int64(len(data.List)) - 1) * commissions.MultisendDelta)
}

func (data MultisendData) Run(tx *Transaction, context *state.State, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.Coins.GetCoin(tx.GasCoin)

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.Reserve().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	if err := checkBalances(context, sender, data.List, commission, tx.GasCoin); err != nil {
		return Response{
			Code: code.InsufficientFunds,
			Log:  err.Error(),
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.Coins.SubVolume(tx.GasCoin, commission)
		context.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		context.Accounts.SubBalance(sender, tx.GasCoin, commission)
		for _, item := range data.List {
			context.Accounts.SubBalance(sender, item.Coin, item.Value)
			context.Accounts.AddBalance(item.To, item.Coin, item.Value)
		}
		context.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeMultisend)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.to"), Value: []byte(pluckRecipients(data.List))},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}

func checkBalances(context *state.State, sender types.Address, items []MultisendDataItem, commission *big.Int, gasCoin types.CoinSymbol) error {
	total := map[types.CoinSymbol]*big.Int{}
	total[gasCoin] = big.NewInt(0).Set(commission)

	for _, item := range items {
		if total[item.Coin] == nil {
			total[item.Coin] = big.NewInt(0)
		}

		total[item.Coin].Add(total[item.Coin], item.Value)
	}

	coins := make([]types.CoinSymbol, 0, len(total))
	for k := range total {
		coins = append(coins, k)
	}

	sort.SliceStable(coins, func(i, j int) bool {
		return coins[i].Compare(coins[j]) == 1
	})

	for _, coin := range coins {
		value := total[coin]
		if context.Accounts.GetBalance(sender, coin).Cmp(value) < 0 {
			return errors.New(fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value, coin))
		}
	}

	return nil
}

func checkCoins(context *state.State, items []MultisendDataItem) error {
	for _, item := range items {
		if !context.Coins.Exists(item.Coin) {
			return errors.New(fmt.Sprintf("Coin %s not exists", item.Coin))
		}
	}

	return nil
}

func pluckRecipients(items []MultisendDataItem) string {
	recipients := make([]string, len(items))
	for i, item := range items {
		recipients[i] = hex.EncodeToString(item.To[:])
	}

	return strings.Join(recipients, ",")
}
