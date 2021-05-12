package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"strconv"
)

type AddLimit struct {
	CoinToSell types.CoinID
	SellVolume *big.Int
	CoinToBuy  types.CoinID
	BuyVolume  *big.Int
}

func (data AddLimit) Gas() int64 {
	return 10
}
func (data AddLimit) TxType() TxType {
	return 0
}

func (data AddLimit) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToBuy.String(),
				data.CoinToSell.String(), "", "")),
		}
	}

	if !context.Swap().SwapPoolExist(data.CoinToSell, data.CoinToBuy) {
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

func (data AddLimit) String() string {
	return fmt.Sprintf("ADD LIMIT")
}

func (data AddLimit) CommissionData(price *commission.Price) *big.Int {
	return big.NewInt(999)
}

func (data AddLimit) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	swapper := checkState.Swap().GetSwapper(data.CoinToBuy, data.CoinToSell)
	if !swapper.Exists() {
		return Response{
			Code: code.PairNotExists,
			Log:  "swap pool for pair not found",
			Info: EncodeError(code.NewPairNotExists(data.CoinToBuy.String(), data.CoinToSell.String())),
		}
	}

	commissionInBaseCoin := tx.Commission(price)
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
	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, data.SellVolume)
		orderID := deliverState.Swap.PairAddSellLimit(data.CoinToSell, data.CoinToBuy, data.SellVolume, data.BuyVolume)

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
