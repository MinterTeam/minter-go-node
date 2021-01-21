package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
	"strconv"
)

type BuySwapPoolData struct {
	CoinToBuy          types.CoinID
	ValueToBuy         *big.Int
	CoinToSell         types.CoinID
	MaximumValueToSell *big.Int
}

func (data BuySwapPoolData) TxType() TxType {
	return TypeBuySwapPool
}

func (data BuySwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.CoinToSell == data.CoinToBuy {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.CoinToSell.String(), "",
				data.CoinToBuy.String(), "")),
		}
	}
	if !context.Swap().SwapPoolExist(data.CoinToSell, data.CoinToBuy) {
		return &Response{
			Code: code.PairNotExists,
			Log:  "swap pool not found",
			Info: EncodeError(code.NewPairNotExists(data.CoinToSell.String(), data.CoinToBuy.String())),
		}
	}
	return nil
}

func (data BuySwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL BUY")
}

func (data BuySwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return price.Convert
}

func (data BuySwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int, gas int64) Response {
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

	commissionInBaseCoin := tx.CommissionInBaseCoin(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	swapper := checkState.Swap().GetSwapper(data.CoinToSell, data.CoinToBuy)
	if isGasCommissionFromPoolSwap {
		if tx.GasCoin == data.CoinToSell && data.CoinToBuy.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
		}
		if tx.GasCoin == data.CoinToBuy && data.CoinToSell.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
		}
	}
	errResp = CheckSwap(swapper, checkState.Coins().GetCoin(data.CoinToSell), checkState.Coins().GetCoin(data.CoinToBuy), data.MaximumValueToSell, data.ValueToBuy, true)
	if errResp != nil {
		return *errResp
	}

	calculatedAmountToSell := swapper.CalculateSellForBuy(data.ValueToBuy)
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

	returnValue := calculatedAmountToSell
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		amountIn, amountOut := deliverState.Swap.PairBuy(data.CoinToSell, data.CoinToBuy, data.MaximumValueToSell, data.ValueToBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, amountOut)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		returnValue = amountIn
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.gas"), Value: []byte(strconv.Itoa(int(gas)))},
		kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeBuySwapPool)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
		kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
		kv.Pair{Key: []byte("tx.return"), Value: []byte(returnValue.String())},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   gas,
		GasWanted: gas,
		Tags:      tags,
	}
}

func CheckSwap(rSwap swap.EditableChecker, coinIn calculateCoin, coinOut calculateCoin, valueIn *big.Int, valueOut *big.Int, isBuy bool) *Response {
	if isBuy {
		calculatedAmountToSell := rSwap.CalculateSellForBuy(valueOut)
		if calculatedAmountToSell == nil {
			reserve0, reserve1 := rSwap.Reserves()
			symbolIn := coinIn.GetFullSymbol()
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but swap pool has reserves %s %s and %s %s", valueIn, symbolIn, valueOut, symbolOut, reserve0.String(), symbolIn, reserve1.String(), symbolOut),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if calculatedAmountToSell.Cmp(valueIn) == 1 {
			return &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s, but currently you need to spend %s to complete tx",
					valueIn.String(), calculatedAmountToSell.String()),
				Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), calculatedAmountToSell.String(), coinIn.GetFullSymbol(), coinIn.ID().String())),
			}
		}
		valueIn = calculatedAmountToSell
	} else {
		calculatedAmountToBuy := rSwap.CalculateBuyForSell(valueIn)
		if calculatedAmountToBuy == nil {
			reserve0, reserve1 := rSwap.Reserves()
			symbolIn := coinIn.GetFullSymbol()
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but swap pool has reserves %s %s and %s %s", valueIn, symbolIn, valueOut, symbolOut, reserve0.String(), symbolIn, reserve1.String(), symbolOut),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if calculatedAmountToBuy.Cmp(valueOut) == -1 {
			return &Response{
				Code: code.MinimumValueToBuyReached,
				Log: fmt.Sprintf(
					"You wanted to buy minimum %s, but currently you need to spend %s to complete tx",
					valueIn.String(), calculatedAmountToBuy.String()),
				Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), calculatedAmountToBuy.String(), coinIn.GetFullSymbol(), coinIn.ID().String())),
			}
		}
		valueOut = calculatedAmountToBuy
	}
	if err := rSwap.CheckSwap(valueIn, valueOut); err != nil {
		if err == swap.ErrorK {
			panic(swap.ErrorK)
		}
		if err == swap.ErrorInsufficientLiquidity {
			reserve0, reserve1 := rSwap.Reserves()
			symbolIn := coinIn.GetFullSymbol()
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but pool reserve %s equal %s and reserve %s equal %s", valueIn, symbolIn, valueOut, symbolOut, reserve0.String(), symbolIn, reserve1.String(), symbolOut),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if err == swap.ErrorInsufficientOutputAmount {
			return &Response{
				Code: code.InsufficientOutputAmount,
				Log:  fmt.Sprintf("Enter a positive number of coins to exchange"),
				Info: EncodeError(code.NewInsufficientOutputAmount(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String())),
			}
		}
		return &Response{
			Code: code.SwapPoolUnknown,
			Log:  err.Error(),
		}
	}
	return nil
}
