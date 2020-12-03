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

type AddSwapPool struct {
	Coin0      types.CoinID
	Coin1      types.CoinID
	Amount0    *big.Int
	MaxAmount1 *big.Int
}

func (data AddSwapPool) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin1 == data.Coin0 {
		return &Response{
			Code: 999,
			Log:  "identical coin",
			// Info: EncodeError(),
		}
	}

	coin0 := context.Coins().GetCoin(data.Coin0)
	if coin0 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin0.String())),
		}
	}

	coin1 := context.Coins().GetCoin(data.Coin1)
	if coin1 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin1.String())),
		}
	}

	if err := context.Swap().CheckMint(data.Coin0, data.Coin1, data.Amount0, data.MaxAmount1); err != nil {
		return &Response{
			Code: 999,
			Log:  err.Error(),
			// Info: EncodeError(),
		}
	}
	return nil
}

func (data AddSwapPool) String() string {
	return fmt.Sprintf("ADD SWAP POOL")
}

func (data AddSwapPool) Gas() int64 {
	return commissions.AddSwapPoolData
}

func (data AddSwapPool) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	amount0 := new(big.Int).Set(data.Amount0)
	if tx.GasCoin == data.Coin0 {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.Coin0).Cmp(amount0) == -1 {
		return Response{Code: code.InsufficientFunds} // todo
	}

	amount1 := new(big.Int).Set(data.MaxAmount1)
	if tx.GasCoin == data.Coin1 {
		amount0.Add(amount1, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.Coin1).Cmp(amount1) == -1 {
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
		amount0, amount1 := deliverState.Swap.PairMint(sender, data.Coin0, data.Coin1, data.Amount0, data.MaxAmount1)

		deliverState.Accounts.SubBalance(sender, data.Coin0, amount0)
		deliverState.Accounts.SubBalance(sender, data.Coin1, amount1)

		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(tx.GasCoin, commission)
		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeAddSwapPool)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
