package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type BurnTokenData struct {
	Coin  types.CoinID
	Value *big.Int
}

func (data BurnTokenData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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

func (data BurnTokenData) String() string {
	return fmt.Sprintf("BURN COIN: %d", data.Coin)
}

func (data BurnTokenData) Gas() int64 {
	return commissions.EditEmissionData
}

func (data BurnTokenData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, priceCoin types.CoinID, price *big.Int) Response {
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

	commissionInBaseCoin := tx.CommissionInBaseCoin()
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

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(data.Coin, data.Value)
		deliverState.Accounts.SubBalance(sender, data.Coin, data.Value)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeBurnToken)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
