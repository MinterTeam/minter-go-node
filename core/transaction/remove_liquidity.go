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
)

type RemoveLiquidity struct {
	Coin0          types.CoinID
	Coin1          types.CoinID
	Liquidity      *big.Int
	MinimumVolume0 *big.Int
	MinimumVolume1 *big.Int
}

func (data RemoveLiquidity) TxType() TxType {
	return TypeRemoveLiquidity
}

func (data RemoveLiquidity) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin0 == data.Coin1 {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.Coin0.String(),
				data.Coin1.String(), "", "")),
		}
	}

	return nil
}

func (data RemoveLiquidity) String() string {
	return fmt.Sprintf("REMOVE SWAP POOL")
}

func (data RemoveLiquidity) CommissionData(price *commission.Price) *big.Int {
	return price.RemoveLiquidity
}

func (data RemoveLiquidity) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	swapper := checkState.Swap().GetSwapper(data.Coin0, data.Coin1)
	if !swapper.IsExist() {
		return Response{
			Code: code.PairNotExists,
			Log:  "swap pool for pair not found",
			Info: EncodeError(code.NewPairNotExists(data.Coin0.String(), data.Coin1.String())),
		}
	}

	coinLiquidity := checkState.Coins().GetCoinBySymbol(LiquidityCoinSymbol(swapper.CoinID()), 0)
	balance := checkState.Accounts().GetBalance(sender, coinLiquidity.ID())
	if balance.Cmp(data.Liquidity) == -1 {
		amount0, amount1 := swapper.Amounts(balance, coinLiquidity.Volume())
		symbol1 := checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()
		symbol0 := checkState.Coins().GetCoin(data.Coin0).GetFullSymbol()
		return Response{
			Code: code.InsufficientLiquidityBalance,
			Log:  fmt.Sprintf("Insufficient balance for provider: %s liquidity tokens is equal %s %s and %s %s, but you want to get %s liquidity", balance, amount0, symbol0, amount1, symbol1, data.Liquidity),
			Info: EncodeError(code.NewInsufficientLiquidityBalance(balance.String(), amount0.String(), data.Coin0.String(), amount1.String(), data.Coin1.String(), data.Liquidity.String())),
		}
	}

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if swapper.IsExist() {
		if isGasCommissionFromPoolSwap {
			if tx.GasCoin == data.Coin0 && data.Coin1.IsBaseCoin() {
				swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
			}
			if tx.GasCoin == data.Coin1 && data.Coin0.IsBaseCoin() {
				swapper = swapper.AddLastSwapStep(commissionInBaseCoin, commission)
			}
		}
	}

	if err := swapper.CheckBurn(data.Liquidity, data.MinimumVolume0, data.MinimumVolume1, coinLiquidity.Volume()); err != nil {
		wantAmount0, wantAmount1 := swapper.Amounts(data.Liquidity, coinLiquidity.Volume())
		if err == swap.ErrorInsufficientLiquidityBurned {
			wantGetAmount0 := data.MinimumVolume0.String()
			wantGetAmount1 := data.MinimumVolume1.String()
			symbol0 := checkState.Coins().GetCoin(data.Coin0).GetFullSymbol()
			symbol1 := checkState.Coins().GetCoin(data.Coin1).GetFullSymbol()
			return Response{
				Code: code.InsufficientLiquidityBurned,
				Log:  fmt.Sprintf("You wanted to get more %s %s and more %s %s, but currently liquidity %s is equal %s %s and %s %s", wantGetAmount0, symbol0, wantGetAmount1, symbol1, data.Liquidity, wantAmount0, symbol0, wantAmount1, symbol1),
				Info: EncodeError(code.NewInsufficientLiquidityBurned(wantGetAmount0, data.Coin0.String(), wantGetAmount1, data.Coin1.String(), data.Liquidity.String(), wantAmount0.String(), wantAmount1.String())),
			}
		}
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}
	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		amount0, amount1 := deliverState.Swap.PairBurn(data.Coin0, data.Coin1, data.Liquidity, data.MinimumVolume0, data.MinimumVolume1, coinLiquidity.Volume())
		deliverState.Accounts.AddBalance(sender, data.Coin0, amount0)
		deliverState.Accounts.AddBalance(sender, data.Coin1, amount1)

		deliverState.Coins.SubVolume(coinLiquidity.ID(), data.Liquidity)
		deliverState.Accounts.SubBalance(sender, coinLiquidity.ID(), data.Liquidity)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
			{Key: []byte("tx.volume0"), Value: []byte(amount0.String())},
			{Key: []byte("tx.volume1"), Value: []byte(amount1.String())},
			{Key: []byte("tx.pool_token"), Value: []byte(coinLiquidity.GetFullSymbol())},
			{Key: []byte("tx.pool_token_id"), Value: []byte(coinLiquidity.ID().String())},
			{Key: []byte("tx.pair_ids"), Value: []byte(liquidityCoinName(data.Coin0, data.Coin1))},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
