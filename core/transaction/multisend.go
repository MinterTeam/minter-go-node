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
	quantity := len(data.List)
	if quantity < 1 || quantity > 100 {
		return &Response{
			Code: code.InvalidMultisendData,
			Log:  "List length must be between 1 and 100",
			Info: EncodeError(map[string]string{
				"code":         fmt.Sprintf("%d", code.InvalidMultisendData),
				"description":  "invalid_multisend_data",
				"min_quantity": "1",
				"max_quantity": "100",
				"got_quantity": fmt.Sprintf("%d", quantity),
			}),
		}
	}

	if errResp := checkCoins(context, data.List); errResp != nil {
		return errResp
	}
	return nil
}

type MultisendDataItem struct {
	Coin  types.CoinSymbol
	To    types.Address
	Value *big.Int
}

func (item MultisendDataItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Coin  string `json:"coin"`
		To    string `json:"to"`
		Value string `json:"value"`
	}{
		Coin:  item.Coin.String(),
		To:    item.To.String(),
		Value: item.Value.String(),
	})
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

		errResp := CheckReserveUnderflow(coin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.Reserve().String(), commissionInBaseCoin.String()),
				Info: EncodeError(map[string]string{
					"has_coin":      coin.Reserve().String(),
					"required_coin": commissionInBaseCoin.String(),
				}),
			}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	if errResp := checkBalances(context, sender, data.List, commission, tx.GasCoin); errResp != nil {
		return *errResp
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

func checkBalances(context *state.State, sender types.Address, items []MultisendDataItem, commission *big.Int, gasCoin types.CoinSymbol) *Response {
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
			return &Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value, coin),
				Info: EncodeError(map[string]string{
					"sender":       sender.String(),
					"needed_value": fmt.Sprintf("%d", value),
					"coin":         fmt.Sprintf("%d", coin),
				}),
			}
		}
	}

	return nil
}

func checkCoins(context *state.State, items []MultisendDataItem) *Response {
	for _, item := range items {
		if !context.Coins.Exists(item.Coin) {
			return &Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin %s not exists", item.Coin),
				Info: EncodeError(map[string]string{
					"coin": fmt.Sprintf("%s", item.Coin),
				}),
			}
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
