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

type RemoveLimitOrderData struct {
	ID uint32
}

func (data RemoveLimitOrderData) Gas() int64 {
	return gasRemoveLimitOrder
}
func (data RemoveLimitOrderData) TxType() TxType {
	return TypeRemoveLimitOrder
}

func (data RemoveLimitOrderData) basicCheck(tx *Transaction, context *state.CheckState) *Response {

	return nil
}

func (data RemoveLimitOrderData) String() string {
	return fmt.Sprintf("REMOVE ORDER")
}

func (data RemoveLimitOrderData) CommissionData(price *commission.Price) *big.Int {
	return price.RemoveLimitOrderPrice()
}

func (data RemoveLimitOrderData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := price
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	order := checkState.Swap().GetOrder(data.ID)
	if order == nil {
		return Response{
			Code: code.OrderNotExists,
			Log:  "limit order not found",
			Info: EncodeError(code.NewOrderNotExists(data.ID)),
		}
	}

	if order.Owner.Compare(sender) != 0 {
		return Response{
			Code: code.IsNotOwnerOfOrder,
			Log:  "Sender is not owner of this order",
			Info: EncodeError(code.NewIsNotOwnerOfOrder(
				order.Coin0.String(),
				order.Coin1.String(),
				data.ID,
				order.Owner.String())),
		}
	}

	swapper := checkState.Swap().GetSwapper(order.Coin0, order.Coin1)
	if isGasCommissionFromPoolSwap && swapper.GetID() == commissionPoolSwapper.GetID() {
		if tx.GasCoin == order.Coin0 && order.Coin1.IsBaseCoin() {
			swapper = swapper.AddLastSwapStepWithOrders(commission, commissionInBaseCoin, true)
		}
		if tx.GasCoin == order.Coin1 && order.Coin0.IsBaseCoin() {
			swapper = swapper.AddLastSwapStepWithOrders(big.NewInt(0).Neg(commissionInBaseCoin), big.NewInt(0).Neg(commission), true)
		}
	}

	if swapper.IsOrderAlreadyUsed(data.ID) {
		return Response{
			Code: code.OrderNotExists,
			Log:  "this limit order will be canceled upon payment of the commission on this transaction",
			Info: EncodeError(code.NewOrderNotExists(data.ID)),
		}
	}

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  []*swap.OrderDetail
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.CommissionCoin(), types.GetBaseCoinID(), commission, big.NewInt(0))
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  ownersCom,
			}
			for _, value := range ownersCom {
				deliverState.Accounts.AddBalance(value.Owner, tx.CommissionCoin(), value.ValueBigInt)
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.CommissionCoin(), commission)
			deliverState.Coins.SubReserve(tx.CommissionCoin(), commissionInBaseCoin)
		}
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		coin, volume := deliverState.Swap.PairRemoveLimitOrder(data.ID)
		if volume.Sign() == 0 {
			panic("order already used")
		}
		deliverState.Accounts.AddBalance(sender, coin, volume)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.order_id"), Value: []byte(strconv.Itoa(int(data.ID)))},
			// {Key: []byte("tx.pair_ids"), Value: []byte(liquidityCoinName(data.Coin0, data.Coin1))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
