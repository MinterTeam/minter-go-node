package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type EditCoinOwnerData struct {
	Symbol   types.CoinSymbol
	NewOwner types.Address
}

func (data EditCoinOwnerData) Gas() int {
	return gasEditCoinOwner
}
func (data EditCoinOwnerData) TxType() TxType {
	return TypeEditCoinOwner
}

func (data EditCoinOwnerData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	sender, _ := tx.Sender()

	if !context.Coins().ExistsBySymbol(data.Symbol) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Symbol),
			Info: EncodeError(code.NewCoinNotExists(data.Symbol.String(), "")),
		}
	}

	info := context.Coins().GetSymbolInfo(data.Symbol)
	if info == nil {
		return &Response{
			Code: code.IsNotOwnerOfCoin,
			Log:  fmt.Sprintf("Sender is not owner of coin"),
			Info: EncodeError(code.NewIsNotOwnerOfCoin(data.Symbol.String(), nil)),
		}
	}

	if info.OwnerAddress() == nil || *info.OwnerAddress() != sender {
		owner := info.OwnerAddress().String()
		return &Response{
			Code: code.IsNotOwnerOfCoin,
			Log:  "Sender is not owner of coin",
			Info: EncodeError(code.NewIsNotOwnerOfCoin(data.Symbol.String(), &owner)),
		}
	}

	return nil
}

func (data EditCoinOwnerData) String() string {
	return fmt.Sprintf("EDIT OWNER COIN symbol:%s new owner:%s", data.Symbol.String(), data.NewOwner.String())
}

func (data EditCoinOwnerData) CommissionData(price *commission.Price) *big.Int {
	return price.EditTickerOwner
}

func (data EditCoinOwnerData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}
	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Coins.ChangeOwner(data.Symbol, data.NewOwner)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_symbol"), Value: []byte(data.Symbol.String()), Index: true},
			{Key: []byte("tx.coin_id"), Value: []byte(checkState.Coins().GetCoinBySymbol(data.Symbol, 0).ID().String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
