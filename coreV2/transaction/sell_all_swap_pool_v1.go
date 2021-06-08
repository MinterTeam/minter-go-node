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

type SellAllSwapPoolDataV1 struct {
	Coins             []types.CoinID
	MinimumValueToBuy *big.Int
}

func (data *SellAllSwapPoolDataV1) Gas() int64 {
	return gasSellAllSwapPool + int64(len(data.Coins)-2)*convertDelta
}

func (data *SellAllSwapPoolDataV1) TxType() TxType {
	return TypeSellAllSwapPool
}

func (data SellAllSwapPoolDataV1) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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

func (data SellAllSwapPoolDataV1) String() string {
	return fmt.Sprintf("SWAP POOL SELL ALL")
}

func (data SellAllSwapPoolDataV1) CommissionData(price *commission.Price) *big.Int {
	return new(big.Int).Add(price.SellAllPoolBase, new(big.Int).Mul(price.SellAllPoolDelta, big.NewInt(int64(len(data.Coins))-2)))
}

func (data SellAllSwapPoolDataV1) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	coinToSell := data.Coins[0]

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(coinToSell, types.GetBaseCoinID())
	sellCoin := checkState.Coins().GetCoin(coinToSell)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, sellCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	balance := checkState.Accounts().GetBalance(sender, coinToSell)
	available := big.NewInt(0).Set(balance)
	balance.Sub(available, commission)

	if balance.Sign() != 1 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), balance.String(), sellCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), balance.String(), sellCoin.GetFullSymbol(), coinToSell.String())),
		}
	}
	lastIteration := len(data.Coins[1:]) - 1
	{
		coinToSell := data.Coins[0]
		coinToSellModel := sellCoin
		valueToSell := big.NewInt(0).Set(balance)
		valueToBuy := big.NewInt(0)
		for i, coinToBuy := range data.Coins[1:] {
			swapper := checkState.Swap().GetSwapper(coinToSell, coinToBuy)
			if isGasCommissionFromPoolSwap == true && coinToBuy.IsBaseCoin() {
				swapper = commissionPoolSwapper.AddLastSwapStep(commission, commissionInBaseCoin)
			}

			if i == lastIteration {
				valueToBuy = data.MinimumValueToBuy
			}

			coinToBuyModel := checkState.Coins().GetCoin(coinToBuy)
			errResp = CheckSwapV230(swapper, coinToSellModel, coinToBuyModel, valueToSell, valueToBuy, false)
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

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(sellCoin.ID(), types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !sellCoin.ID().IsBaseCoin() {
			deliverState.Coins.SubVolume(sellCoin.ID(), commission)
			deliverState.Coins.SubReserve(sellCoin.ID(), commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, sellCoin.ID(), commission)

		coinToSell := data.Coins[0]
		valueToSell := big.NewInt(0).Set(balance)

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

		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		amountOut := valueToSell

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.coin_to_buy"), Value: []byte(data.Coins[len(data.Coins)-1].String()), Index: true},
			{Key: []byte("tx.coin_to_sell"), Value: []byte(data.Coins[0].String()), Index: true},
			{Key: []byte("tx.return"), Value: []byte(amountOut.String())},
			{Key: []byte("tx.sell_amount"), Value: []byte(available.String())},
			{Key: []byte("tx.pools"), Value: []byte(poolIDs.string())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
