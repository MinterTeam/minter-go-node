package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type SellSwapPoolData struct {
	CoinToSell        types.CoinID
	ValueToSell       *big.Int
	CoinToBuy         types.CoinID
	MinimumValueToBuy *big.Int
}

func (data SellSwapPoolData) TxType() TxType {
	return TypeSellSwapPool
}

func (data SellSwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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
			Log:  fmt.Sprint("swap pair not exists in pool"),
			Info: EncodeError(code.NewPairNotExists(data.CoinToSell.String(), data.CoinToBuy.String())),
		}
	}
	return nil
}

func (data SellSwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL SELL")
}

func (data SellSwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return price.SellPool
}

func (data SellSwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	swapper := checkState.Swap().GetSwapper(data.CoinToSell, data.CoinToBuy)
	if isGasCommissionFromPoolSwap {
		if tx.GasCoin == data.CoinToSell && data.CoinToBuy.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
		}
		if tx.GasCoin == data.CoinToBuy && data.CoinToSell.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
		}
	}
	errResp = CheckSwap(swapper, checkState.Coins().GetCoin(data.CoinToSell), checkState.Coins().GetCoin(data.CoinToBuy), data.ValueToSell, data.MinimumValueToBuy, false)
	if errResp != nil {
		return *errResp
	}

	amount0 := new(big.Int).Set(data.ValueToSell)
	if tx.GasCoin != data.CoinToSell {
		if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
			}
		}
	} else {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.CoinToSell).Cmp(amount0) == -1 {
		symbol := checkState.Coins().GetCoin(data.CoinToSell).GetFullSymbol()
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), symbol),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), amount0.String(), symbol, data.CoinToSell.String())),
		}
	}

	var tags kv.Pairs
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		amountIn, amountOut := deliverState.Swap.PairSell(data.CoinToSell, data.CoinToBuy, data.ValueToSell, data.MinimumValueToBuy)
		deliverState.Accounts.SubBalance(sender, data.CoinToSell, amountIn)
		deliverState.Accounts.AddBalance(sender, data.CoinToBuy, amountOut)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = kv.Pairs{
			kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			kv.Pair{Key: []byte("tx.coin_to_buy"), Value: []byte(data.CoinToBuy.String())},
			kv.Pair{Key: []byte("tx.coin_to_sell"), Value: []byte(data.CoinToSell.String())},
			kv.Pair{Key: []byte("tx.return"), Value: []byte(amountOut.String())},
			kv.Pair{Key: []byte("tx.pair_ids"), Value: []byte(liquidityCoinName(data.CoinToBuy, data.CoinToSell))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
