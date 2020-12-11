package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/swap"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type AddSwapPoolData struct {
	Coin0          types.CoinID
	Coin1          types.CoinID
	Volume0        *big.Int
	MaximumVolume1 *big.Int
}

func (data AddSwapPoolData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Coin1 == data.Coin0 {
		return &Response{
			Code: code.CrossConvert,
			Log:  "\"From\" coin equals to \"to\" coin",
			Info: EncodeError(code.NewCrossConvert(
				data.Coin0.String(),
				data.Coin1.String(), "", "")),
		}
	}

	coin0 := context.Coins().GetCoin(data.Coin0)
	if coin0 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin0.String())),
		}
	}

	coin1 := context.Coins().GetCoin(data.Coin1)
	if coin1 == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", data.Coin1.String())),
		}
	}

	if err := context.Swap().CheckMint(data.Coin0, data.Coin1, data.Volume0, data.MaximumVolume1); err != nil {
		if err == swap.ErrorInsufficientLiquidityMinted {
			if !context.Swap().SwapPoolExist(data.Coin0, data.Coin1) {
				return &Response{
					Code: code.InsufficientLiquidityMinted,
					Log: fmt.Sprintf("You wanted to add less than minimum liquidity, you should add %s of coin %d and %s or more of coin %d",
						"10", data.Coin0, "10", data.Coin1),
					Info: EncodeError(code.NewInsufficientLiquidityMinted(data.Coin0.String(), "10", data.Coin1.String(), "10")),
				}
			} else {
				amount0, amount1 := context.Swap().AmountsOfLiquidity(data.Coin0, data.Coin1, big.NewInt(1))
				return &Response{
					Code: code.InsufficientLiquidityMinted,
					Log: fmt.Sprintf("You wanted to add less than one liquidity, you should add %s of coin %d and %s or more of coin %d",
						amount0, data.Coin0, amount1, data.Coin1),
					Info: EncodeError(code.NewInsufficientLiquidityMinted(data.Coin0.String(), amount0.String(), data.Coin1.String(), amount1.String())),
				}
			}
		} else if err == swap.ErrorInsufficientInputAmount {
			_, _, neededAmount1, _ := context.Swap().PairCalculateAddLiquidity(data.Coin0, data.Coin1, data.Volume0)
			return &Response{
				Code: code.InsufficientInputAmount,
				Log:  fmt.Sprintf("You wanted to add %s of coin %d, but currently you need to add %s of coin %d to complete tx", data.Volume0, data.Coin0, neededAmount1, data.Coin1),
				Info: EncodeError(code.NewInsufficientInputAmount(data.Coin0.String(), data.Volume0.String(), data.Coin1.String(), data.MaximumVolume1.String(), neededAmount1.String())),
			}
		}
	}
	return nil
}

func (data AddSwapPoolData) String() string {
	return fmt.Sprintf("ADD SWAP POOL")
}

func (data AddSwapPoolData) Gas() int64 {
	return commissions.AddSwapPoolData
}

func (data AddSwapPoolData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	amount0 := new(big.Int).Set(data.Volume0)
	if tx.GasCoin == data.Coin0 {
		amount0.Add(amount0, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.Coin0).Cmp(amount0) == -1 {
		return Response{Code: code.InsufficientFunds} // todo
	}

	amount1 := new(big.Int).Set(data.MaximumVolume1)
	if tx.GasCoin == data.Coin1 {
		amount0.Add(amount1, commission)
	}
	if checkState.Accounts().GetBalance(sender, data.Coin1).Cmp(amount1) == -1 {
		return Response{Code: code.InsufficientFunds} // todo
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		amount0, amount1 := deliverState.Swap.PairMint(sender, data.Coin0, data.Coin1, data.Volume0, data.MaximumVolume1)

		deliverState.Accounts.SubBalance(sender, data.Coin0, amount0)
		deliverState.Accounts.SubBalance(sender, data.Coin1, amount1)

		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeAddSwapPool)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
