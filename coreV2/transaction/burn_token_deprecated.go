package transaction

import (
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type BurnTokenDataDeprecated struct {
	Coin  types.CoinID
	Value *big.Int
}

func (data BurnTokenDataDeprecated) Gas() int64 {
	return gasBurnToken
}
func (data BurnTokenDataDeprecated) TxType() TxType {
	return TypeBurnToken
}

func (data BurnTokenDataDeprecated) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	coin := context.Coins().GetCoin(data.Coin)
	if coin == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !coin.IsBurnable() {
		return &Response{
			Code: code.CoinNotBurnable,
			Log:  "Coin not burnable",
			Info: EncodeError(code.NewCoinIsNotBurnable(coin.GetFullSymbol(), data.Coin.String())),
		}
	}

	if big.NewInt(0).Sub(coin.Volume(), data.Value).Cmp(minTokenSupply) == -1 {
		return &Response{
			Code: code.WrongCoinEmission,
			Log:  fmt.Sprintf("Coin volume should be more than %s", minTokenSupply),
			Info: EncodeError(code.NewWrongCoinEmission(minTokenSupply.String(), coin.MaxSupply().String(), coin.Volume().String(), "", data.Value.String())),
		}
	}

	// todo: remove owner check
	sender, _ := tx.Sender()
	symbolInfo := context.Coins().GetSymbolInfo(coin.Symbol())
	if coin.Version() != 0 || symbolInfo == nil || symbolInfo.OwnerAddress().Compare(sender) != 0 {
		var owner *string
		if symbolInfo != nil && symbolInfo.OwnerAddress() != nil {
			own := symbolInfo.OwnerAddress().String()
			owner = &own
		}
		return &Response{
			Code: code.IsNotOwnerOfCoin,
			Log:  "Sender is not owner of coin",
			Info: EncodeError(code.NewIsNotOwnerOfCoin(coin.Symbol().String(), owner)),
		}
	}

	return nil
}

func (data BurnTokenDataDeprecated) String() string {
	return fmt.Sprintf("BURN COIN: %d", data.Coin)
}

func (data BurnTokenDataDeprecated) CommissionData(price *commission.Price) *big.Int {
	return price.BurnToken
}

func (data BurnTokenDataDeprecated) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	value := big.NewInt(0).Set(data.Value)
	if tx.GasCoin == data.Coin {
		value.Add(value, commission)
	}

	if checkState.Accounts().GetBalance(sender, data.Coin).Cmp(value) == -1 {
		symbol := checkState.Coins().GetCoin(data.Coin).GetFullSymbol()
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), value.String(), symbol),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), value.String(), symbol, data.Coin.String())),
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
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(data.Coin, data.Value)
		deliverState.Accounts.SubBalance(sender, data.Coin, data.Value)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_id"), Value: []byte(data.Coin.String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
