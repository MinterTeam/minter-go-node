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

type BuySwapPoolData struct {
	CoinToBuy          types.CoinID
	ValueToBuy         *big.Int
	CoinToSell         types.CoinID
	MaximumValueToSell *big.Int
}

func (data BuySwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	response := CheckSwap(context, data.CoinToSell, data.MaximumValueToSell, data.CoinToBuy, data.ValueToBuy, true)
	if response != nil {
		return response
	}
	return nil
}

func (data BuySwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL BUY")
}

func (data BuySwapPoolData) Gas() int64 {
	return commissions.ConvertTx
}

func (data BuySwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	calculatedAmountToSell, _ := checkState.Swap().PairCalculateSellForBuy(data.CoinToSell, data.CoinToBuy, data.ValueToBuy)
	amount0 := new(big.Int).Set(calculatedAmountToSell)
	if tx.GasCoin == data.CoinToSell {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.CoinToSell).Cmp(amount0) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), checkState.Coins().GetCoin(data.CoinToSell).GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), amount0.String(), checkState.Coins().GetCoin(data.CoinToSell).GetFullSymbol(), data.CoinToSell.String())),
		}
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	returnValue := data.MaximumValueToSell
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

		returnValue = amountIn
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeBuySwapPool)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		kv.Pair{Key: []byte("tx.return"), Value: []byte(returnValue.String())},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}

func CheckSwap(context *state.CheckState, coinIn types.CoinID, valueIn *big.Int, coinOut types.CoinID, valueOut *big.Int, isBuy bool) *Response {
	rSwap := context.Swap()
	if coinIn == coinOut {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				coinIn.String(), "",
				coinOut.String(), "")),
		}
	}
	if !context.Swap().SwapPoolExist(coinIn, coinOut) {
		return &Response{
			Code: code.PairNotExists,
			Log:  fmt.Sprintf("swap pair %d %d not exists in pool", coinIn, coinOut),
			Info: EncodeError(code.NewPairNotExists(coinIn.String(), coinOut.String())),
		}
	}
	if isBuy {
		calculatedAmountToSell, err := context.Swap().PairCalculateSellForBuy(coinIn, coinOut, valueOut)
		if err == swap.ErrorInsufficientLiquidity {
			_, reserve0, reserve1 := rSwap.SwapPool(coinIn, coinOut)
			symbolIn := context.Coins().GetCoin(coinIn).GetFullSymbol()
			symbolOut := context.Coins().GetCoin(coinOut).GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but pool reserve %s equal %s and reserve %s equal %s", symbolIn, valueIn, symbolOut, valueOut, symbolIn, reserve0.String(), symbolOut, reserve1.String()),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.String(), valueIn.String(), coinOut.String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if err != nil {
			return &Response{
				Code: code.SwapPoolUnknown,
				Log:  err.Error(),
			}
		}
		if calculatedAmountToSell.Cmp(valueIn) == 1 {
			coin := context.Coins().GetCoin(coinIn)
			return &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					valueIn.String(), calculatedAmountToSell.String()),
				Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), calculatedAmountToSell.String(), coin.GetFullSymbol(), coin.ID().String())),
			}
		}
		valueIn = calculatedAmountToSell
	} else {
		calculatedAmountToBuy, err := rSwap.PairCalculateBuyForSell(coinIn, coinOut, valueIn)
		if err == swap.ErrorInsufficientLiquidity {
			_, reserve0, reserve1 := rSwap.SwapPool(coinIn, coinOut)
			symbolIn := context.Coins().GetCoin(coinIn).GetFullSymbol()
			symbolOut := context.Coins().GetCoin(coinOut).GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but pool reserve %s equal %s and reserve %s equal %s", symbolIn, valueIn, symbolOut, valueOut, symbolIn, reserve0.String(), symbolOut, reserve1.String()),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.String(), valueIn.String(), coinOut.String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if err != nil {
			return &Response{
				Code: code.SwapPoolUnknown,
				Log:  err.Error(),
			}
		}
		if calculatedAmountToBuy.Cmp(valueOut) == -1 {
			coin := context.Coins().GetCoin(coinIn)
			return &Response{
				Code: code.MinimumValueToBuyReached,
				Log: fmt.Sprintf(
					"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
					valueIn.String(), calculatedAmountToBuy.String()),
				Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), calculatedAmountToBuy.String(), coin.GetFullSymbol(), coin.ID().String())),
			}
		}
		valueOut = calculatedAmountToBuy
	}
	if err := rSwap.CheckSwap(coinIn, coinOut, valueIn, valueOut); err != nil {
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
					Code: code.MinimumValueToBuyReached,
					Log: fmt.Sprintf(
						"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
						valueIn.String(), value.String()),
					Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), value.String(), coin.GetFullSymbol(), coin.ID().String())),
				}
			}
		}
		if err == swap.ErrorInsufficientLiquidity {
			_, reserve0, reserve1 := rSwap.SwapPool(coinIn, coinOut)
			symbolIn := context.Coins().GetCoin(coinIn).GetFullSymbol()
			symbolOut := context.Coins().GetCoin(coinOut).GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but pool reserve %s equal %s and reserve %s equal %s", symbolIn, valueIn, symbolOut, valueOut, symbolIn, reserve0.String(), symbolOut, reserve1.String()),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.String(), valueIn.String(), coinOut.String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if err == swap.ErrorInsufficientOutputAmount {
			return &Response{
				Code: code.InsufficientOutputAmount,
				Log:  fmt.Sprintf("Enter a positive number of coins to exchange"),
				Info: EncodeError(code.NewInsufficientOutputAmount(coinIn.String(), valueIn.String(), coinOut.String(), valueOut.String())),
			}
		}
		return &Response{
			Code: code.SwapPoolUnknown,
			Log:  err.Error(),
		}
	}
	return nil
}
