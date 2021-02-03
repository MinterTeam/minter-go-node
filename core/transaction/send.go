package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type SendData struct {
	Coin  types.CoinID
	To    types.Address
	Value *big.Int
}

func (data SendData) TxType() TxType {
	return TypeSend
}

type Coin struct {
	ID     uint32 `json:"id"`
	Symbol string `json:"symbol"`
}

func (data SendData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Value == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if !context.Coins().Exists(data.Coin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Coin),
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	return nil
}

func (data SendData) String() string {
	return fmt.Sprintf("SEND to:%s coin:%s value:%s",
		data.To.String(), data.Coin.String(), data.Value.String())
}

func (data SendData) CommissionData(price *commission.Price) *big.Int {
	return price.Send
}

func (data SendData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	needValue := big.NewInt(0).Set(commission)
	if tx.GasCoin == data.Coin {
		needValue.Add(data.Value, needValue)
	} else {
		if checkState.Accounts().GetBalance(sender, data.Coin).Cmp(data.Value) < 0 {
			coin := checkState.Coins().GetCoin(data.Coin)
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.Value.String(), coin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), data.Value.String(), coin.GetFullSymbol(), coin.ID().String())),
			}
		}
	}
	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(needValue) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), needValue.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), needValue.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}
	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SubBalance(sender, data.Coin, data.Value)
		deliverState.Accounts.AddBalance(data.To, data.Coin, data.Value)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(data.To[:]))},
			{Key: []byte("tx.coin_id"), Value: []byte(data.Coin.String())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
