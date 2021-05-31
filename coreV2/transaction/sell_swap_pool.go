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

type SellSwapPoolData struct {
	Coins             []types.CoinID
	ValueToSell       *big.Int
	MinimumValueToBuy *big.Int
}

func (data SellSwapPoolData) TxType() TxType {
	return TypeSellSwapPool
}

func (data SellSwapPoolData) Gas() int64 {
	return gasSellSwapPool + int64(len(data.Coins)-2)*convertDelta
}

func (data SellSwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if len(data.Coins) < 2 {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}
	if len(data.Coins) > 5 {
		return &Response{
			Code: code.TooLongSwapRoute,
			Log:  "maximum allowed length of the exchange chain is 5",
			Info: EncodeError(code.NewCustomCode(code.TooLongSwapRoute)),
		}
	}
	coin0 := data.Coins[0]
	for _, coin1 := range data.Coins[1:] {
		if coin0 == coin1 {
			return &Response{
				Code: code.CrossConvert,
				Log:  "\"From\" coin equals to \"to\" coin",
				Info: EncodeError(code.NewCrossConvert(
					coin0.String(), "",
					coin1.String(), "")),
			}
		}
		if !context.Swap().SwapPoolExist(coin0, coin1) {
			return &Response{
				Code: code.PairNotExists,
				Log:  fmt.Sprint("swap pool not exists"),
				Info: EncodeError(code.NewPairNotExists(coin0.String(), coin1.String())),
			}
		}
		coin0 = coin1
	}

	return nil
}

func (data SellSwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL SELL")
}

func (data SellSwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return new(big.Int).Add(price.SellPoolBase, new(big.Int).Mul(price.SellPoolDelta, big.NewInt(int64(len(data.Coins))-2)))
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

	lastIteration := len(data.Coins[1:]) - 1
	{
		checkDuplicatePools := map[uint32]struct{}{}
		coinToSell := data.Coins[0]
		coinToSellModel := checkState.Coins().GetCoin(coinToSell)
		// resultCoin := data.Coins[lastIteration]
		valueToSell := data.ValueToSell
		valueToBuy := big.NewInt(0)
		for i, coinToBuy := range data.Coins[1:] {
			swapper := checkState.Swap().GetSwapper(coinToSell, coinToBuy)
			if _, ok := checkDuplicatePools[swapper.GetID()]; ok {
				return Response{
					Code: code.DuplicatePoolInRoute,
					Log:  fmt.Sprintf("Forbidden to repeat the pool in the route, pool duplicate %d", swapper.GetID()),
					Info: EncodeError(code.NewDuplicatePoolInRouteCode(swapper.GetID())),
				}
			}
			checkDuplicatePools[swapper.GetID()] = struct{}{}
			if isGasCommissionFromPoolSwap {
				if tx.GasCoin == coinToSell && coinToBuy.IsBaseCoin() {
					swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
				}
				if tx.GasCoin == coinToBuy && coinToSell.IsBaseCoin() {
					swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
				}
			}

			if i == lastIteration {
				valueToBuy = data.MinimumValueToBuy
			}

			coinToBuyModel := checkState.Coins().GetCoin(coinToBuy)
			errResp = CheckSwap(swapper, coinToSellModel, coinToBuyModel, valueToSell, valueToBuy, false)
			if errResp != nil {
				return *errResp
			}

			valueToSellCalc := swapper.CalculateBuyForSell(valueToSell)
			if valueToSellCalc == nil {
				reserve0, reserve1 := swapper.Reserves()
				return Response{
					Code: code.SwapPoolUnknown,
					Log:  fmt.Sprintf("swap pool has reserves %s %s and %d %s, you wanted sell %s %s", reserve0, coinToSellModel.GetFullSymbol(), reserve1, coinToBuyModel.GetFullSymbol(), valueToSell, coinToSellModel.GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientLiquidity(coinToSellModel.ID().String(), valueToSell.String(), coinToBuyModel.ID().String(), valueToSellCalc.String(), reserve0.String(), reserve1.String())),
				}
			}
			valueToSell = valueToSellCalc
			coinToSellModel = coinToBuyModel
			coinToSell = coinToBuy
		}
	}

	coinToSell := data.Coins[0]
	amount0 := new(big.Int).Set(data.ValueToSell)
	if tx.GasCoin != coinToSell {
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
	if checkState.Accounts().GetBalance(sender, coinToSell).Cmp(amount0) == -1 {
		symbol := checkState.Coins().GetCoin(coinToSell).GetFullSymbol()
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), symbol),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), amount0.String(), symbol, coinToSell.String())),
		}
	}

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		coinToSell := data.Coins[0]
		valueToSell := data.ValueToSell

		var poolIDs tagPoolsChange

		for i, coinToBuy := range data.Coins[1:] {
			amountIn, amountOut, poolID := deliverState.Swap.PairSell(coinToSell, coinToBuy, valueToSell, big.NewInt(0))

			poolIDs = append(poolIDs, &tagPoolChange{
				PoolID:   poolID,
				CoinIn:   coinToSell,
				ValueIn:  amountIn.String(),
				CoinOut:  coinToBuy,
				ValueOut: amountOut.String(),
			})

			if i == 0 {
				deliverState.Accounts.SubBalance(sender, coinToSell, amountIn)
			}

			valueToSell = amountOut
			coinToSell = coinToBuy

			if i == lastIteration {
				deliverState.Accounts.AddBalance(sender, coinToBuy, amountOut)
			}
		}

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		amountOut := valueToSell
		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.Coins[len(data.Coins)-1].String()), Index: true},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.Coins[0].String()), Index: true},
			{Key: []byte("tx.return"), Value: []byte(amountOut.String())},
			{Key: []byte("tx.pools"), Value: []byte(poolIDs.string())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
