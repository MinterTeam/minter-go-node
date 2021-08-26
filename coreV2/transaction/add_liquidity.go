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

type AddLiquidityData struct {
	Coin0          types.CoinID
	Coin1          types.CoinID
	Volume0        *big.Int
	MaximumVolume1 *big.Int
}

func (data AddLiquidityData) Gas() int64 {
	return gasAddLiquidity
}

func (data AddLiquidityData) TxType() TxType {
	return TypeAddLiquidity
}

func (data AddLiquidityData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin1 == data.Coin0 {
		return &Response{
			Code: code.CrossConvert,
			Log:  "First coin equals to second coin",
			Info: EncodeError(code.NewCrossConvert(
				data.Coin0.String(),
				data.Coin1.String(), "", "")),
		}
	}

	if !context.Swap().SwapPoolExist(data.Coin0, data.Coin1) {
		return &Response{
			Code: code.PairNotExists,
			Log:  "swap pool not found",
			Info: EncodeError(code.NewPairNotExists(
				data.Coin0.String(),
				data.Coin1.String())),
		}
	}

	return nil
}

func (data AddLiquidityData) String() string {
	return fmt.Sprintf("ADD SWAP POOL")
}

func (data AddLiquidityData) CommissionData(price *commission.Price) *big.Int {
	return price.AddLiquidity
}

func (data AddLiquidityData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	neededAmount1 := new(big.Int).Set(data.MaximumVolume1)

	swapper := checkState.Swap().GetSwapper(data.Coin0, data.Coin1)
	if isGasCommissionFromPoolSwap && swapper.GetID() == commissionPoolSwapper.GetID() {
		if tx.GasCoin == data.Coin0 && data.Coin1.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
		}
		if tx.GasCoin == data.Coin1 && data.Coin0.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
		}
	}
	coinLiquidity := checkState.Coins().GetCoinBySymbol(LiquidityCoinSymbol(swapper.GetID()), 0)
	_, neededAmount1 = swapper.CalculateAddLiquidity(data.Volume0, coinLiquidity.Volume())
	if neededAmount1.Cmp(data.MaximumVolume1) == 1 {
		return Response{
			Code: code.InsufficientInputAmount,
			Log:  fmt.Sprintf("You wanted to add %s %s, but currently you need to add %s %s to complete tx", data.Volume0, checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), neededAmount1, checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientInputAmount(data.Coin0.String(), data.Volume0.String(), data.Coin1.String(), data.MaximumVolume1.String(), neededAmount1.String())),
		}
	}

	if err := swapper.CheckMint(data.Volume0, neededAmount1, coinLiquidity.Volume()); err != nil {
		if err == swap.ErrorInsufficientLiquidityMinted {
			amount0, amount1 := swapper.Amounts(big.NewInt(1), coinLiquidity.Volume())
			return Response{
				Code: code.InsufficientLiquidityMinted,
				Log: fmt.Sprintf("You wanted to add less than one liquidity, you should add %s %s and %s %s or more",
					amount0, checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), amount1, checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientLiquidityMinted(data.Coin0.String(), amount0.String(), data.Coin1.String(), amount1.String())),
			}
		} else if err == swap.ErrorInsufficientInputAmount {
			return Response{
				Code: code.InsufficientInputAmount,
				Log:  fmt.Sprintf("You wanted to add %s %s, but currently you need to add %s %s to complete tx", data.Volume0, checkState.Coins().GetCoin(data.Coin0).GetFullSymbol(), neededAmount1, checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientInputAmount(data.Coin0.String(), data.Volume0.String(), data.Coin1.String(), data.MaximumVolume1.String(), neededAmount1.String())),
			}
		}
	}
	{
		amount0 := new(big.Int).Set(data.Volume0)
		if tx.GasCoin == data.Coin0 {
			amount0.Add(amount0, commission)
		}
		if checkState.Accounts().GetBalance(sender, data.Coin0).Cmp(amount0) == -1 {
			symbol := checkState.Coins().GetCoin(data.Coin0).GetFullSymbol()
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), amount0.String(), symbol),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), amount0.String(), symbol, data.Coin0.String())),
			}
		}
	}

	{
		maximumVolume1 := new(big.Int).Set(neededAmount1)
		if tx.GasCoin == data.Coin1 {
			maximumVolume1.Add(maximumVolume1, commission)
		}
		if checkState.Accounts().GetBalance(sender, data.Coin1).Cmp(maximumVolume1) == -1 {
			symbol := checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), maximumVolume1.String(), symbol),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), maximumVolume1.String(), symbol, data.Coin1.String())),
			}
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
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairBuyWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  ownersCom,
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

		amount0, amount1, liquidity := deliverState.Swap.PairMint(data.Coin0, data.Coin1, data.Volume0, data.MaximumVolume1, coinLiquidity.Volume())
		deliverState.Accounts.SubBalance(sender, data.Coin0, amount0)
		deliverState.Accounts.SubBalance(sender, data.Coin1, amount1)

		deliverState.Coins.AddVolume(coinLiquidity.ID(), liquidity)
		deliverState.Accounts.AddBalance(sender, coinLiquidity.ID(), liquidity)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.volume1"), Value: []byte(amount1.String())},
			{Key: []byte("tx.liquidity"), Value: []byte(liquidity.String())},
			{Key: []byte("tx.pool_token"), Value: []byte(coinLiquidity.GetFullSymbol()), Index: true},
			{Key: []byte("tx.pool_token_id"), Value: []byte(coinLiquidity.ID().String()), Index: true},
			{Key: []byte("tx.pair_ids"), Value: []byte(liquidityCoinName(data.Coin0, data.Coin1))},
			{Key: []byte("tx.pool_id"), Value: []byte(types.CoinID(swapper.GetID()).String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}

func liquidityCoinName(c0, c1 types.CoinID) string {
	if c0 < c1 {
		return fmt.Sprintf("%d-%d", c0, c1)
	}
	return fmt.Sprintf("%d-%d", c1, c0)
}
