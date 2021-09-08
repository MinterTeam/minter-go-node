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

type AddLimitOrderData struct {
	CoinToSell  types.CoinID
	ValueToSell *big.Int
	CoinToBuy   types.CoinID
	ValueToBuy  *big.Int
}

func (data AddLimitOrderData) Gas() int64 {
	return gasAddLimitOrder
}
func (data AddLimitOrderData) TxType() TxType {
	return TypeAddLimitOrder
}

func (data AddLimitOrderData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToBuy.String(),
				data.CoinToSell.String(), "", "")),
		}
	}

	if data.ValueToBuy.Sign() != 1 || data.ValueToSell.Sign() != 1 {
		return &Response{
			Code: code.DecodeError,
			Log:  "positive values of coin volumes are expected",
			Info: EncodeError(code.NewDecodeError()),
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

	return nil
}

func (data AddLimitOrderData) String() string {
	return fmt.Sprintf("ADD ORDER")
}

func (data AddLimitOrderData) CommissionData(price *commission.Price) *big.Int {
	return price.AddLimitOrderPrice()
}

func (data AddLimitOrderData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	swapper := checkState.Swap().GetSwapper(data.CoinToSell, data.CoinToBuy)
	if isGasCommissionFromPoolSwap && swapper.GetID() == commissionPoolSwapper.GetID() {
		if tx.GasCoin == data.CoinToSell && data.CoinToBuy.IsBaseCoin() {
			swapper = swapper.AddLastSwapStepWithOrders(commission, commissionInBaseCoin, true)
		}
		if tx.GasCoin == data.CoinToBuy && data.CoinToSell.IsBaseCoin() {
			swapper = swapper.AddLastSwapStepWithOrders(big.NewInt(0).Neg(commissionInBaseCoin), big.NewInt(0).Neg(commission), true)
		}
	}
	currentPrice := swapper.Reverse().Price()
	maxPrice := big.NewFloat(0).Quo(currentPrice, big.NewFloat(5))
	orderPrice := swap.CalcPriceSell(data.ValueToBuy, data.ValueToSell)
	if currentPrice.Cmp(orderPrice) == -1 ||
		maxPrice.Cmp(orderPrice) == 1 {
		return Response{
			Code: code.WrongOrderPrice,
			Log:  fmt.Sprintf("order price is %s, but must be less than %s and more than %s", orderPrice.Text('f', 18), currentPrice.Text('f', 18), maxPrice.Text('f', 18)),
			Info: EncodeError(code.NewWrongOrderPrice(currentPrice.Text('f', 18), maxPrice.Text('f', 18), orderPrice.Text('f', 18))),
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
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, data.ValueToSell)
		orderID, poolID := deliverState.Swap.PairAddOrder(data.CoinToBuy, data.CoinToSell, data.ValueToBuy, data.ValueToSell, sender, currentBlock)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.pool_id"), Value: []byte(strconv.Itoa(int(poolID)))},
			{Key: []byte("tx.order_id"), Value: []byte(strconv.Itoa(int(orderID)))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
