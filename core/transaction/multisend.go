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
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
	"sort"
	"strings"
)

type MultisendData struct {
	List []MultisendDataItem
}

type MultisendDataItem struct {
	Coin  types.CoinSymbol
	To    types.Address
	Value *big.Int
}

func (data MultisendData) MarshalJSON() ([]byte, error) {
	var list []interface{}

	for _, item := range data.List {
		list = append(list, struct {
			Coin  types.CoinSymbol
			To    types.Address
			Value string
		}{
			Coin:  item.Coin,
			To:    item.To,
			Value: item.Value.String(),
		})
	}

	return json.Marshal(list)
}

func (data MultisendData) String() string {
	return fmt.Sprintf("MULTISEND")
}

func (data MultisendData) Gas() int64 {
	return commissions.SendTx + ((int64(len(data.List)) - 1) * commissions.MultisendDelta)
}

func (data MultisendData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response {
	if len(data.List) < 1 || len(data.List) > 100 {
		return Response{
			Code: code.InvalidMultisendData,
			Log:  "List length must be between 1 and 100"}
	}

	if err := checkCoins(context, data.List); err != nil {
		return Response{
			Code: code.CoinNotExists,
			Log:  err.Error()}
	}

	if !context.CoinExists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if err := checkBalances(context, sender, data.List, commission, tx.GasCoin); err != nil {
		return Response{
			Code: code.InsufficientFunds,
			Log:  err.Error(),
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		if !tx.GasCoin.IsBaseCoin() {
			context.SubCoinVolume(tx.GasCoin, commission)
			context.SubCoinReserve(tx.GasCoin, commissionInBaseCoin)
		}

		context.SubBalance(sender, tx.GasCoin, commission)
		for _, item := range data.List {
			context.SubBalance(sender, item.Coin, item.Value)
			context.AddBalance(item.To, item.Coin, item.Value)
		}
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeMultisend}},
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

func checkBalances(context *state.StateDB, sender types.Address, items []MultisendDataItem, commission *big.Int, gasCoin types.CoinSymbol) error {
	total := map[types.CoinSymbol]*big.Int{}
	total[gasCoin] = commission

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

	sort.Slice(coins, func(i, j int) bool {
		return coins[i].Compare(coins[j]) == 1
	})

	for _, coin := range coins {
		value := total[coin]
		if context.GetBalance(sender, coin).Cmp(value) < 0 {
			return errors.New(fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value, coin))
		}
	}

	return nil
}

func checkCoins(context *state.StateDB, items []MultisendDataItem) error {
	for _, item := range items {
		if !context.CoinExists(item.Coin) {
			return errors.New(fmt.Sprintf("Coin %s not exists", item.Coin))
		}
	}

	return nil
}

func pluckRecipients(items []MultisendDataItem) string {
	var recipients []string

	for _, item := range items {
		recipients = append(recipients, hex.EncodeToString(item.To[:]))
	}

	return strings.Join(recipients, ",")
}
