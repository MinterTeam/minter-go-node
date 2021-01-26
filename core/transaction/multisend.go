package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
)

type MultisendData struct {
	List []MultisendDataItem `json:"list"`
}

func (data MultisendData) TxType() TxType {
	return TypeMultisend
}

func (data MultisendData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	quantity := len(data.List)
	if quantity < 1 || quantity > 100 {
		return &Response{
			Code: code.InvalidMultisendData,
			Log:  "List length must be between 1 and 100",
			Info: EncodeError(code.NewInvalidMultisendData("1", "100", fmt.Sprintf("%d", quantity))),
		}
	}

	if errResp := checkCoins(context, data.List); errResp != nil {
		return errResp
	}
	return nil
}

type MultisendDataItem struct {
	Coin  types.CoinID
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
	return "MULTISEND"
}

func (data MultisendData) CommissionData(price *commission.Price) *big.Int {
	return big.NewInt(0).Add(price.MultisendBase, big.NewInt(0).Mul(big.NewInt(int64(len(data.List))-1), price.MultisendDelta))
}

func (data MultisendData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int, gas int64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if errResp := checkBalances(checkState, sender, data.List, commission, tx.GasCoin); errResp != nil {
		return *errResp
	}

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		for _, item := range data.List {
			deliverState.Accounts.SubBalance(sender, item.Coin, item.Value)
			deliverState.Accounts.AddBalance(item.To, item.Coin, item.Value)
		}
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.gas"), Value: []byte(strconv.Itoa(int(gas)))},
		kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeMultisend)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.to"), Value: []byte(pluckRecipients(data.List))},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   gas,
		GasWanted: gas,
	}
}

func checkBalances(context *state.CheckState, sender types.Address, items []MultisendDataItem, commission *big.Int, gasCoin types.CoinID) *Response {
	total := map[types.CoinID]*big.Int{}
	total[gasCoin] = big.NewInt(0).Set(commission)

	for _, item := range items {
		if total[item.Coin] == nil {
			total[item.Coin] = big.NewInt(0)
		}

		total[item.Coin].Add(total[item.Coin], item.Value)
	}

	coins := make([]types.CoinID, 0, len(total))
	for k := range total {
		coins = append(coins, k)
	}

	sort.SliceStable(coins, func(i, j int) bool {
		return coins[i] > coins[j]
	})

	for _, coin := range coins {
		value := total[coin]
		coinData := context.Coins().GetCoin(coin)
		if context.Accounts().GetBalance(sender, coin).Cmp(value) < 0 {
			return &Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value, coinData.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), value.String(), coinData.GetFullSymbol(), coinData.ID().String())),
			}
		}
	}

	return nil
}

func checkCoins(context *state.CheckState, items []MultisendDataItem) *Response {
	for _, item := range items {
		if !context.Coins().Exists(item.Coin) {
			return &Response{
				Code: code.CoinNotExists,
				Log:  fmt.Sprintf("Coin %s not exists", item.Coin),
				Info: EncodeError(code.NewCoinNotExists("", item.Coin.String())),
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
