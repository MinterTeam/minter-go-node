package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
	"github.com/MinterTeam/minter-go-node/core/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
	"strings"
)

type BuySwapPoolData struct {
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

func (data BuySwapPoolData) Gas() int {
	return gasBuySwapPool
}
func (data BuySwapPoolData) TxType() TxType {
	return TypeBuySwapPool
}

func (data BuySwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if len(data.Coins) < 2 {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
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

func (data BuySwapPoolData) String() string {
	return fmt.Sprintf("SWAP POOL BUY")
}

func (data BuySwapPoolData) CommissionData(price *commission.Price) *big.Int {
	return new(big.Int).Add(price.BuyPoolBase, new(big.Int).Mul(price.BuyPoolDelta, big.NewInt(int64(len(data.Coins))-2)))
}

func (data BuySwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	reverseCoinIds(data.Coins)

	var calculatedAmountToSell *big.Int
	resultCoin := data.Coins[len(data.Coins)-1]
	{
		coinToBuy := data.Coins[0]
		coinToBuyModel := checkState.Coins().GetCoin(coinToBuy)
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)
		valueToSell := maxCoinSupply
		for _, coinToSell := range data.Coins[1:] {
			swapper := checkState.Swap().GetSwapper(coinToSell, coinToBuy)
			if isGasCommissionFromPoolSwap {
				if tx.GasCoin == coinToSell && coinToBuy.IsBaseCoin() {
					swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
				}
				if tx.GasCoin == coinToSell && coinToBuy.IsBaseCoin() {
					swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
				}
			}

			if coinToSell == resultCoin {
				valueToSell = data.MaximumValueToSell
			}

			coinToSellModel := checkState.Coins().GetCoin(coinToSell)
			errResp = CheckSwap(swapper, coinToSellModel, coinToBuyModel, valueToSell, valueToBuy, true)
			if errResp != nil {
				return *errResp
			}

			valueToBuy = swapper.CalculateSellForBuy(valueToBuy)
			if valueToBuy == nil {
				reserve0, reserve1 := swapper.Reserves()
				return Response{
					Code: code.SwapPoolUnknown,
					Log:  fmt.Sprintf("swap pool has reserves %s %s and %d %s, you wanted buy %s %s", reserve0, coinToSellModel.GetFullSymbol(), reserve1, coinToBuyModel.GetFullSymbol(), valueToBuy, coinToBuyModel.GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
				}
			}
			coinToBuyModel = coinToSellModel
			coinToBuy = coinToSell
		}
		calculatedAmountToSell = valueToBuy
	}

	coinToSell := resultCoin
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
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		coinToBuy := data.Coins[0]
		resultCoin := data.Coins[len(data.Coins)-1]
		valueToBuy := data.ValueToBuy

		var poolIDs []string

		for i, coinToSell := range data.Coins[1:] {

			amountIn, amountOut, poolID := deliverState.Swap.PairBuy(coinToSell, coinToBuy, maxCoinSupply, valueToBuy)

			poolIDs = append(poolIDs, fmt.Sprintf("%d:%d-%s:%d-%s", poolID, coinToSell, amountIn.String(), coinToBuy, amountOut.String()))

			if i == 0 {
				deliverState.Accounts.AddBalance(sender, coinToBuy, amountOut)
			}

			valueToBuy = amountIn
			coinToBuy = coinToSell

			if coinToSell == resultCoin {
				deliverState.Accounts.SubBalance(sender, coinToSell, amountIn)
			}
		}
		amountIn := valueToBuy

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.Coins[0].String())},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(resultCoin.String())},
			{Key: []byte("tx.return"), Value: []byte(amountIn.String())},
			{Key: []byte("tx.pools"), Value: []byte(strings.Join(poolIDs, ","))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func CheckSwap(rSwap swap.EditableChecker, coinIn CalculateCoin, coinOut CalculateCoin, valueIn *big.Int, valueOut *big.Int, isBuy bool) *Response {
	if isBuy {
		calculatedAmountToSell := rSwap.CalculateSellForBuy(valueOut)
		if calculatedAmountToSell == nil {
			reserve0, reserve1 := rSwap.Reserves()
			symbolIn := coinIn.GetFullSymbol()
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.InsufficientLiquidity,
				Log:  fmt.Sprintf("You wanted to buy %s %s, but swap pool has reserve %s %s", valueOut, symbolOut, reserve0.String(), symbolIn),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if calculatedAmountToSell.Cmp(valueIn) == 1 {
			return &Response{
				Code: code.MaximumValueToSellReached,
				Log: fmt.Sprintf(
					"You wanted to sell maximum %s %s, but currently you need to spend %s %s to complete tx",
					valueIn.String(), coinIn.GetFullSymbol(), calculatedAmountToSell.String(), coinOut.GetFullSymbol()),
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
				Log:  fmt.Sprintf("You wanted to sell %s %s and get more than the swap pool has a reserve in %s", valueIn, symbolIn, symbolOut),
				Info: EncodeError(code.NewInsufficientLiquidity(coinIn.ID().String(), valueIn.String(), coinOut.ID().String(), valueOut.String(), reserve0.String(), reserve1.String())),
			}
		}
		if calculatedAmountToBuy.Cmp(valueOut) == -1 {
			symbolOut := coinOut.GetFullSymbol()
			return &Response{
				Code: code.MinimumValueToBuyReached,
				Log: fmt.Sprintf(
					"You wanted to buy minimum %s %s, but currently you buy only %s %s",
					valueIn.String(), symbolOut, calculatedAmountToBuy.String(), symbolOut),
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
