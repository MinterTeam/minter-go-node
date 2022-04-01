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

type BuySwapPoolDataV260 struct {
	Coins              []types.CoinID
	ValueToBuy         *big.Int
	MaximumValueToSell *big.Int
}

func reverseCoinIds(a []types.CoinID) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}

func reversePools(a []*tagPoolChange) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}

func (data BuySwapPoolDataV260) Gas() int64 {
	return gasBuySwapPool + int64(len(data.Coins)-2)*convertDelta
}
func (data BuySwapPoolDataV260) TxType() TxType {
	return TypeBuySwapPool
}

func (data BuySwapPoolDataV260) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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

func (data BuySwapPoolDataV260) String() string {
	return fmt.Sprintf("SWAP POOL BUY")
}

func (data BuySwapPoolDataV260) CommissionData(price *commission.Price) *big.Int {
	return new(big.Int).Add(price.BuyPoolBase, new(big.Int).Mul(price.BuyPoolDelta, big.NewInt(int64(len(data.Coins))-2)))
}

func (data BuySwapPoolDataV260) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	reverseCoinIds(data.Coins)

	var calculatedAmountToSell *big.Int
	lastIteration := len(data.Coins[1:]) - 1
	{
		checkDuplicatePools := map[uint32]struct{}{}
		coinToBuy := data.Coins[0]
		coinToBuyModel := checkState.Coins().GetCoin(coinToBuy)
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)
		valueToSell := maxCoinSupply
		for i, coinToSell := range data.Coins[1:] {
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
				valueToSell = data.MaximumValueToSell
			}

			coinToSellModel := checkState.Coins().GetCoin(coinToSell)
			var valueToSellCalc *big.Int
			errResp, valueToSellCalc, _ = CheckSwap(swapper, coinToSellModel, coinToBuyModel, valueToSell, valueToBuy, true)
			if errResp != nil {
				return *errResp
			}

			if valueToSellCalc == nil || valueToSellCalc.Sign() != 1 {
				reserve0, reserve1 := swapper.Reserves()
				return Response{
					Code: code.InsufficientLiquidity,
					Log:  fmt.Sprintf("swap pool has reserves %s %s and %d %s, you wanted buy %s %s", reserve0, coinToSellModel.GetFullSymbol(), reserve1, coinToBuyModel.GetFullSymbol(), valueToBuy, coinToSellModel.GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientLiquidity(coinToSellModel.ID().String(), "", coinToBuyModel.ID().String(), valueToBuy.String(), reserve0.String(), reserve1.String())),
				}
			}

			valueToBuy = valueToSellCalc
			coinToBuyModel = coinToSellModel
			coinToBuy = coinToSell
		}
		calculatedAmountToSell = valueToBuy
	}

	coinToSell := data.Coins[len(data.Coins)-1]
	amount0 := new(big.Int).Set(calculatedAmountToSell)
	if tx.GasCoin == coinToSell {
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

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) == -1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
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

		coinToBuy := data.Coins[0]
		valueToBuy := data.ValueToBuy

		var poolIDs tagPoolsChange

		for i, coinToSell := range data.Coins[1:] {
			amountIn, amountOut, poolID, details, owners := deliverState.Swapper().PairBuyWithOrders(coinToSell, coinToBuy, maxCoinSupply, valueToBuy)

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
				deliverState.Accounts.AddBalance(sender, coinToBuy, amountOut)
			}

			valueToBuy = amountIn
			coinToBuy = coinToSell

			if i == lastIteration {
				deliverState.Accounts.SubBalance(sender, coinToSell, amountIn)
			}
		}
		reversePools(poolIDs)
		amountIn := valueToBuy

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.Coins[0].String()), Index: true},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.Coins[len(data.Coins)-1].String()), Index: true},
			{Key: []byte("tx.return"), Value: []byte(amountIn.String())},
			{Key: []byte("tx.pools"), Value: []byte(poolIDs.string())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func CheckSwap(rSwap swap.EditableChecker, coinIn CalculateCoin, coinOut CalculateCoin, valueIn *big.Int, valueOut *big.Int, isBuy bool) (resp *Response, res *big.Int, orders []*swap.Limit) {
	if isBuy {
		calculatedAmountToSell, ordrs := rSwap.CalculateSellForBuyWithOrders(valueOut)
		if calculatedAmountToSell == nil {
			reserve0, reserve1 := rSwap.Reserves()
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to buy %s %s, but swap pool has reserve %s %s", valueOut, symbolOut, reserve1.String(), symbolOut),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}, nil, orders
		}
		if calculatedAmountToSell.Cmp(valueIn) == 1 {
			return &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s %s, but currently you need to spend %s %s to complete tx",
					valueIn.String(), coinIn.GetFullSymbol(), calculatedAmountToSell.String(), coinIn.GetFullSymbol()),
				Info: EncodeError(code.NewMaximumValueToSellReached(valueIn.String(), calculatedAmountToSell.String(), coinIn.GetFullSymbol(), coinIn.ID().String())),
			}, nil, orders
		}
		valueIn = calculatedAmountToSell
		res = valueIn
		orders = ordrs
	} else {
		calculatedAmountToBuy, ordrs := rSwap.CalculateBuyForSellWithOrders(valueIn)
		if calculatedAmountToBuy == nil {
			reserve0, reserve1 := rSwap.Reserves()
			symbolIn := coinIn.GetFullSymbol()
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to sell %s %s and get more than the swap pool has a reserve in %s", valueIn, symbolIn, symbolOut),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}, nil, orders
		}
		if valueOut.Sign() == 0 {
			valueOut = big.NewInt(1)
		}
		if calculatedAmountToBuy.Cmp(valueOut) == -1 {
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.MinimumValueToBuyReached,
				Log: fmt.Sprintf(
					"You wanted to buy minimum %s %s, but currently you buy only %s %s",
					valueOut.String(), symbolOut, calculatedAmountToBuy.String(), symbolOut),
				Info: EncodeError(code.NewMinimumValueToBuyReached(valueOut.String(), calculatedAmountToBuy.String(), coinIn.GetFullSymbol(), coinIn.ID().String())),
			}, nil, orders
		}
		valueOut = calculatedAmountToBuy
		res = valueOut
		orders = ordrs
	}

	//_, commission1orders, _, amount1, _ := swap.CalcDiffPool(valueIn, valueOut, orders)
	//_, reserve1 := rSwap.Reserves()
	//if reserve1.Cmp(big.NewInt(0).Sub(amount1, commission1orders)) != 1 {
	//	reserve0, reserve1 := rSwap.Reserves()
	//	symbolIn := coinIn.GetFullSymbol()
	//	symbolOut := coinOut.GetFullSymbol()
	//	r := &Response{
	//		Code: code.InsufficientLiquidity,
	//		Log:  fmt.Sprintf("You wanted to exchange %s %s for %s %s, but the pool reserves are %s %s and %s %s", valueIn, symbolIn, valueOut, symbolOut, reserve0.String(), symbolIn, reserve1.String(), symbolOut),
	//		Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
	//	}
	//	return r, nil, orders
	//}
	return nil, res, orders
}
