package transaction

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type AddOrderSwapPoolData struct {
	CoinToSell  types.CoinID
	ValueToSell *big.Int
	CoinToBuy   types.CoinID
	ValueToBuy  *big.Int
}

func (data AddOrderSwapPoolData) Gas() int64 {
	return 1
}
func (data AddOrderSwapPoolData) TxType() TxType {
	return TypeAddOrderSwapPool
}

func (data AddOrderSwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToBuy.String(),
				data.CoinToSell.String(), "", "")),
		}
	}

	swapper := context.Swap().GetSwapper(data.CoinToSell, data.CoinToBuy)
	if !swapper.Exists() {
		return &Response{
			Code: code.PairNotExists,
			Log:  "swap pool not found",
			Info: EncodeError(code.NewPairNotExists(
				data.CoinToSell.String(),
				data.CoinToBuy.String())),
		}
	}

	if swapper.Price().Cmp(swap.CalcPriceSell(data.ValueToBuy, data.ValueToSell)) == -1 {
		return &Response{
			Code: 123456,
			Log:  "price high",
			Info: EncodeError(code.NewCustomCode(123456)),
		}
	}

	return nil
}

func (data AddOrderSwapPoolData) String() string {
	return fmt.Sprintf("ADD ORDER")
}

func (data AddOrderSwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return price.CreateSwapPool // todo: add new commission's field
}

func (data AddOrderSwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	amountSell := new(big.Int).Set(data.ValueToSell)
	if tx.GasCoin != data.CoinToSell {
		if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
			}
		}
	} else {
		amountSell.Add(amountSell, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.CoinToSell).Cmp(amountSell) < 0 {
		coin := checkState.Coins().GetCoin(data.CoinToSell)
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amountSell.String(), coin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), amountSell.String(), coin.GetFullSymbol(), coin.ID().String())),
		}
	}

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _, _, _ = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, data.ValueToSell)
		orderID := deliverState.Swap.PairAddOrder(data.CoinToBuy, data.CoinToSell, data.ValueToBuy, data.ValueToSell, sender)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.order_id"), Value: []byte(strconv.Itoa(int(orderID)))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
