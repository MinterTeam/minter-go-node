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

type RemoveLiquidityV240 struct {
	Coin0          types.CoinID
	Coin1          types.CoinID
	Liquidity      *big.Int
	MinimumVolume0 *big.Int
	MinimumVolume1 *big.Int
}

func (data RemoveLiquidityV240) Gas() int64 {
	return gasRemoveLiquidity
}
func (data RemoveLiquidityV240) TxType() TxType {
	return TypeRemoveLiquidity
}

func (data RemoveLiquidityV240) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Liquidity.Sign() != 1 {
		return &Response{
			Code: code.DecodeError,
			Log:  "Can't remove zero liquidity volume",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if data.Coin0 == data.Coin1 {
		return &Response{
			Code: code.CrossConvert,
			Log:  "First coin equals to second coin",
			Info: EncodeError(code.NewCrossConvert(
				data.Coin0.String(),
				data.Coin1.String(), "", "")),
		}
	}

	return nil
}

func (data RemoveLiquidityV240) String() string {
	return fmt.Sprintf("REMOVE SWAP POOL")
}

func (data RemoveLiquidityV240) CommissionData(price *commission.Price) *big.Int {
	return price.RemoveLiquidity
}

func (data RemoveLiquidityV240) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	swapper := checkState.Swap().GetSwapper(data.Coin0, data.Coin1)
	if !swapper.Exists() {
		return Response{
			Code: code.PairNotExists,
			Log:  "swap pool for pair not found",
			Info: EncodeError(code.NewPairNotExists(data.Coin0.String(), data.Coin1.String())),
		}
	}

	if isGasCommissionFromPoolSwap && swapper.GetID() == commissionPoolSwapper.GetID() {
		if tx.GasCoin == data.Coin0 && data.Coin1.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(commission, commissionInBaseCoin)
		}
		if tx.GasCoin == data.Coin1 && data.Coin0.IsBaseCoin() {
			swapper = swapper.AddLastSwapStep(big.NewInt(0).Neg(commissionInBaseCoin), big.NewInt(0).Neg(commission))
		}
	}

	coinLiquidity := checkState.Coins().GetCoinBySymbol(LiquidityCoinSymbol(swapper.GetID()), 0)
	balance := checkState.Accounts().GetBalance(sender, coinLiquidity.ID())

	needValue := big.NewInt(0).Set(commission)
	if tx.GasCoin == coinLiquidity.ID() {
		needValue.Add(data.Liquidity, needValue)
	} else {
		if balance.Cmp(data.Liquidity) == -1 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.Liquidity.String(), coinLiquidity.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), data.Liquidity.String(), coinLiquidity.GetFullSymbol(), coinLiquidity.ID().String())),
			}
		}
	}
	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(needValue) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), needValue.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), needValue.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
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

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  map[types.Address]*big.Int
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.GasCoin,
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  make([]*OrderDetail, 0, len(ownersCom)),
			}
			for address, value := range ownersCom {
				deliverState.Accounts.AddBalance(address, tx.GasCoin, value)
				tagsCom.Sellers = append(tagsCom.Sellers, &OrderDetail{Owner: address, Value: value.String()})
			}
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
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.volume0"), Value: []byte(amount0.String())},
			{Key: []byte("tx.volume1"), Value: []byte(amount1.String())},
			{Key: []byte("tx.pool_token"), Value: []byte(coinLiquidity.GetFullSymbol()), Index: true},
			{Key: []byte("tx.pool_token_id"), Value: []byte(coinLiquidity.ID().String()), Index: true},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.pair_ids"), Value: []byte(liquidityCoinName(data.Coin0, data.Coin1))},
			{Key: []byte("tx.pool_id"), Value: []byte(types.CoinID(swapper.GetID()).String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
