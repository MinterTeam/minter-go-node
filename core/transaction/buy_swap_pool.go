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

type BuySwapPool struct {
	CoinSell      types.CoinID
	MaxAmountSell *big.Int
	CoinBuy       types.CoinID
	AmountBuy     *big.Int
}

func (data BuySwapPool) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.CoinBuy == data.CoinSell {
		return &Response{
			Code: 999,
			Log:  "identical coin",
			// Info: EncodeError(),
		}
	}

	if !context.Swap().SwapPoolExist(data.CoinSell, data.CoinBuy) {
		return &Response{
			Code: 999,
			Log:  "swap pool not found",
			// Info: EncodeError(),
		}
	}

	if err := context.Swap().CheckSwap(data.CoinSell, data.CoinBuy, data.MaxAmountSell, data.AmountBuy); err != nil {
		return &Response{
			Code: 999,
			Log:  err.Error(),
			// Info: EncodeError(),
		}
	}
	return nil
}

func (data BuySwapPool) String() string {
	return fmt.Sprintf("EXCHANGE SWAP POOL")
}

func (data BuySwapPool) Gas() int64 {
	return commissions.ConvertTx
}

func (data BuySwapPool) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	amount0 := new(big.Int).Set(data.MaxAmountSell)
	if tx.GasCoin == data.CoinSell {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.CoinSell).Cmp(amount0) == -1 {
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
		amountIn, amountOut := deliverState.Swap.PairBuy(data.CoinSell, data.CoinBuy, data.MaxAmountSell, data.AmountBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinBuy, amountOut)

		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(tx.GasCoin, commission)
		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeBuySwapPool)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
