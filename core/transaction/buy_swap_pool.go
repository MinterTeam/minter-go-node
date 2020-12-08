package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type BuySwapPool struct {
	CoinToBuy          types.CoinID
	ValueToBuy         *big.Int
	CoinToSell         types.CoinID
	MaximumValueToSell *big.Int
}

func (data BuySwapPool) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.CoinToBuy == data.CoinToSell {
		return &Response{
			Code: 999,
			Log:  "identical coin",
			// Info: EncodeError(),
		}
	}

	response := checkSwap(context, data.CoinToSell, data.MaximumValueToSell, data.CoinToBuy, data.ValueToBuy, true)
	if response != nil {
		return response
	}
	return nil
}

func (data BuySwapPool) String() string {
	return fmt.Sprintf("EXCHANGE SWAP POOL: BUY")
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
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	amount0 := new(big.Int).Set(data.MaximumValueToSell)
	if tx.GasCoin == data.CoinToSell {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.CoinToSell).Cmp(amount0) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), checkState.Coins().GetCoin(data.CoinToSell).GetFullSymbol()),
		}
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		amountIn, amountOut := deliverState.Swap.PairBuy(data.CoinToSell, data.CoinToBuy, data.MaximumValueToSell, data.ValueToBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, amountOut)

		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

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

func checkSwap(context *state.CheckState, coinIn types.CoinID, valueIn *big.Int, coinOut types.CoinID, valueOut *big.Int, isBuy bool) *Response {
	rSwap := context.Swap()
	if err := rSwap.CheckSwap(coinIn, coinOut, valueIn, valueOut); err != nil {
		if err == swap.ErrorNotExist {
			return &Response{
				Code: code.PairNotExists,
				Log:  fmt.Sprintf("swap pair %d %d not exists in pool", coinIn, coinOut),
				Info: EncodeError(code.NewPairNotExists(coinIn.String(), coinOut.String())),
			}
		}
		if err == swap.ErrorK {
			if isBuy {
				value, _ := rSwap.PairCalculateBuyForSell(coinIn, coinOut, valueOut)
				coin := context.Coins().GetCoin(coinIn)
				return &Response{
					Code: code.MaximumValueToSellReached,
					Log: fmt.Sprintf(
						"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
						valueIn.String(), value.String()),
					Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), value.String(), coin.GetFullSymbol(), coin.ID().String())),
				}
			} else {
				value, _ := rSwap.PairCalculateSellForBuy(coinIn, coinOut, valueOut)
				coin := context.Coins().GetCoin(coinIn)
				return &Response{
					Code: code.MaximumValueToSellReached,
					Log: fmt.Sprintf(
						"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
						valueIn.String(), value.String()),
					Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), value.String(), coin.GetFullSymbol(), coin.ID().String())),
				}
			}
		}
		if err == swap.ErrorInsufficientLiquidity {
			_, reserve0, reserve1 := rSwap.SwapPool(coinIn, coinOut)
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s pips of coin %d for %s of coin %d, but pool reserve of coin %d equal %s and reserve of coin %d equal %s", coinIn, valueIn, coinOut, valueOut, coinIn, reserve0.String(), coinOut, reserve1.String()),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.String(), valueIn.String(), coinOut.String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		return &Response{
			Code: code.SwapPoolUnknown,
			Log:  err.Error(),
		}
	}
	return nil
}
