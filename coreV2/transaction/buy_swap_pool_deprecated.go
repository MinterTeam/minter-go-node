package transaction

import (
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type BuySwapPoolDataDeprecated struct {
	Coins              []types.CoinID
	ValueToBuy         *big.Int
	MaximumValueToSell *big.Int
}

func (data BuySwapPoolDataDeprecated) Gas() int64 {
	return gasBuySwapPool + int64(len(data.Coins)-2)*convertDelta
}
func (data BuySwapPoolDataDeprecated) TxType() TxType {
	return TypeBuySwapPool
}

func (data BuySwapPoolDataDeprecated) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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

func (data BuySwapPoolDataDeprecated) String() string {
	return fmt.Sprintf("SWAP POOL BUY")
}

func (data BuySwapPoolDataDeprecated) CommissionData(price *commission.Price) *big.Int {
	return new(big.Int).Add(price.BuyPoolBase, new(big.Int).Mul(price.BuyPoolDelta, big.NewInt(int64(len(data.Coins))-2)))
}

func (data BuySwapPoolDataDeprecated) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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
	lastIteration := len(data.Coins[1:]) - 1
	{
		coinToBuy := data.Coins[0]
		coinToBuyModel := checkState.Coins().GetCoin(coinToBuy)
		valueToBuy := big.NewInt(0).Set(data.ValueToBuy)
		valueToSell := maxCoinSupply
		for i, coinToSell := range data.Coins[1:] {
			swapper := checkState.Swap().GetSwapper(coinToSell, coinToBuy)

			if isGasCommissionFromPoolSwap {
				if tx.GasCoin == coinToSell && coinToBuy.IsBaseCoin() {
					swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
				}
				if tx.GasCoin == coinToSell && coinToBuy.IsBaseCoin() {
					swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
				}
			}

			if i == lastIteration {
				valueToSell = data.MaximumValueToSell
			}

			coinToSellModel := checkState.Coins().GetCoin(coinToSell)
			errResp = CheckSwapV230(swapper, coinToSellModel, coinToBuyModel, valueToSell, valueToBuy, true)
			if errResp != nil {
				return *errResp
			}

			valueToBuyCalc := swapper.CalculateSellForBuy(valueToBuy)
			if valueToBuyCalc == nil {
				reserve0, reserve1 := swapper.Reserves()
				return Response{
					Code: code.SwapPoolUnknown,
					Log:  fmt.Sprintf("swap pool has reserves %s %s and %d %s, you wanted buy %s %s", reserve0, coinToSellModel.GetFullSymbol(), reserve1, coinToBuyModel.GetFullSymbol(), valueToBuy, coinToSellModel.GetFullSymbol()),
					Info: EncodeError(code.NewInsufficientLiquidity(coinToSellModel.ID().String(), valueToBuyCalc.String(), coinToBuyModel.ID().String(), valueToBuy.String(), reserve0.String(), reserve1.String())),
				}
			}
			valueToBuy = valueToBuyCalc
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
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		coinToBuy := data.Coins[0]
		valueToBuy := data.ValueToBuy

		var poolIDs tagPoolsChange

		for i, coinToSell := range data.Coins[1:] {

			amountIn, amountOut, poolID := deliverState.Swap.PairBuy(coinToSell, coinToBuy, maxCoinSupply, valueToBuy)

			poolIDs = append(poolIDs, &tagPoolChange{
				PoolID:   poolID,
				CoinIn:   coinToSell,
				ValueIn:  amountIn.String(),
				CoinOut:  coinToBuy,
				ValueOut: amountOut.String(),
			})

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
