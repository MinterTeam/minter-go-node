package transaction

import (
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type SellSwapPoolDataV260 struct {
	Coins             []types.CoinID
	ValueToSell       *big.Int
	MinimumValueToBuy *big.Int
}

func (data SellSwapPoolDataV260) TxType() TxType {
	return TypeSellSwapPool
}

func (data SellSwapPoolDataV260) Gas() int64 {
	return gasSellSwapPool + int64(len(data.Coins)-2)*convertDelta
}

func (data SellSwapPoolDataV260) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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

func (data SellSwapPoolDataV260) String() string {
	return fmt.Sprintf("SWAP POOL SELL")
}

func (data SellSwapPoolDataV260) CommissionData(price *commission.Price) *big.Int {
	return new(big.Int).Add(price.SellPoolBase, new(big.Int).Mul(price.SellPoolDelta, big.NewInt(int64(len(data.Coins))-2)))
}

func (data SellSwapPoolDataV260) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := price
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

			if isGasCommissionFromPoolSwap && swapper.GetID() == commissionPoolSwapper.GetID() {
				commissionInBaseCoin, _ = commissionPoolSwapper.CalculateBuyForSellWithOrders(commission)
				if tx.GasCoin == coinToSell && coinToBuy.IsBaseCoin() {
					swapper = swapper.AddLastSwapStepWithOrders(commission, commissionInBaseCoin, true)
				}
				if tx.GasCoin == coinToBuy && coinToSell.IsBaseCoin() {
					swapper = swapper.AddLastSwapStepWithOrders(big.NewInt(0).Neg(commissionInBaseCoin), big.NewInt(0).Neg(commission), true)
				}
			}

			if i == lastIteration {
				valueToBuy = data.MinimumValueToBuy
			}

			coinToBuyModel := checkState.Coins().GetCoin(coinToBuy)
			var valueToBuyCalc *big.Int
			errResp, valueToBuyCalc, _ = CheckSwap(swapper, coinToSellModel, coinToBuyModel, valueToSell, valueToBuy, false)
			if errResp != nil {
				return *errResp
			}

			if valueToBuyCalc == nil || valueToBuyCalc.Sign() != 1 {
				reserve0, reserve1 := swapper.Reserves()
				return Response{
					Code: code.InsufficientLiquidity,
					Log:  fmt.Sprintf("swap pool has reserves %s %s and %d %s, you wanted sell %s %s", reserve0, coinToSellModel.GetFullSymbol(), reserve1, coinToBuyModel.GetFullSymbol(), valueToSell, coinToSellModel.GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientLiquidity(coinToSellModel.ID().String(), valueToSell.String(), coinToBuyModel.ID().String(), valueToBuyCalc.String(), reserve0.String(), reserve1.String())),
				}
			}
			valueToSell = valueToBuyCalc
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
		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  []*swap.OrderDetail
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swapper().PairSellWithOrders(tx.CommissionCoin(), types.GetBaseCoinID(), commission, big.NewInt(0))
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				// Sellers:  ownersCom,
			}
			for _, value := range ownersCom {
				deliverState.Accounts.AddBalance(value.Owner, tx.CommissionCoin(), value.ValueBigInt)
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.CommissionCoin(), commission)
			deliverState.Coins.SubReserve(tx.CommissionCoin(), commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		coinToSell := data.Coins[0]
		valueToSell := data.ValueToSell

		var poolIDs tagPoolsChange

		for i, coinToBuy := range data.Coins[1:] {
			amountIn, amountOut, poolID, details, owners := deliverState.Swapper().PairSellWithOrders(coinToSell, coinToBuy, valueToSell, big.NewInt(0))

			tags := &tagPoolChange{
				PoolID:   poolID,
				CoinIn:   coinToSell,
				ValueIn:  amountIn.String(),
				CoinOut:  coinToBuy,
				ValueOut: amountOut.String(),
				Orders:   details,
				// Sellers:  owners,
			}

			for _, value := range owners {
				deliverState.Accounts.AddBalance(value.Owner, coinToSell, value.ValueBigInt)
			}
			poolIDs = append(poolIDs, tags)

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
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
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
