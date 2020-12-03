package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type AddExchangeLiquidity struct {
	Coin         types.CoinID
	AmountBase   *big.Int
	AmountCustom *big.Int
}

func (data AddExchangeLiquidity) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin == types.GetSwapHubCoinID() {
		return &Response{
			Code: 999,
			Log:  "identical coin",
			// Info: EncodeError(),
		}
	}
	coin := context.Coins().GetCoin(data.Coin)
	if coin == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if coin.ID() != types.GetBaseCoinID() && coin.HasReserve() {
		return &Response{
			Code: 999,
			Log:  "has reserve",
			// Info: EncodeError(),
		}
	}
	if err := context.Swap().CheckMint(data.Coin, data.AmountBase, data.AmountCustom); err != nil {
		return &Response{
			Code: 999,
			Log:  err.Error(),
			// Info: EncodeError(),
		}
	}
	return nil
}

func (data AddExchangeLiquidity) String() string {
	return fmt.Sprintf("MINT LIQUIDITY")
}

func (data AddExchangeLiquidity) Gas() int64 {
	return commissions.AddExchangeLiquidityData
}

func (data AddExchangeLiquidity) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

	if !tx.GasCoin.IsBaseCoin() {
		errResp := CheckReserveUnderflow(gasCoin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	amount0 := new(big.Int).Set(data.AmountBase)
	if tx.GasCoin == types.GetSwapHubCoinID() {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, types.GetSwapHubCoinID()).Cmp(amount0) == -1 {
		return Response{Code: code.InsufficientFunds} // todo
	}

	amount1 := new(big.Int).Set(data.AmountCustom)
	if tx.GasCoin == data.Coin {
		amount0.Add(amount1, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.Coin).Cmp(amount1) == -1 {
		return Response{Code: code.InsufficientFunds} // todo
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		amount0, amount1 := deliverState.Swap.PairMint(sender, data.Coin, data.AmountBase, data.AmountCustom)

		deliverState.Accounts.SubBalance(sender, types.GetSwapHubCoinID(), amount0)
		deliverState.Accounts.SubBalance(sender, data.Coin, amount1)

		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(tx.GasCoin, commission)
		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeAddExchangeLiquidity)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
