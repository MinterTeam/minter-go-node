package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type MoveStakeData struct {
	From, To types.Pubkey
	Coin     types.CoinID
	Value    *big.Int
}

func (data MoveStakeData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if !context.Coins().Exists(data.Coin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Coin),
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !context.Candidates().Exists(data.From) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with %s public key not found", data.From),
			Info: EncodeError(code.NewCandidateNotFound(data.From.String())),
		}
	}
	if !context.Candidates().Exists(data.To) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with %s public key not found", data.To),
			Info: EncodeError(code.NewCandidateNotFound(data.To.String())),
		}
	}

	sender, _ := tx.Sender()

	if waitlist := context.WaitList().Get(sender, data.From, data.Coin); waitlist != nil {
		if data.Value.Cmp(waitlist.Value) == 1 {
			return &Response{
				Code: code.InsufficientWaitList,
				Log:  "Insufficient amount at waitlist for sender account",
				Info: EncodeError(code.NewInsufficientWaitList(waitlist.Value.String(), data.Value.String())),
			}
		}
	} else {
		stake := context.Candidates().GetStakeValueOfAddress(data.From, sender, data.Coin)

		if stake == nil {
			return &Response{
				Code: code.StakeNotFound,
				Log:  "Stake of current user not found",
				Info: EncodeError(code.NewStakeNotFound(data.From.String(), sender.String(), data.Coin.String(), context.Coins().GetCoin(data.Coin).GetFullSymbol())),
			}
		}

		if stake.Cmp(data.Value) == -1 {
			return &Response{
				Code: code.InsufficientStake,
				Log:  "Insufficient stake for sender account",
				Info: EncodeError(code.NewInsufficientStake(data.From.String(), sender.String(), data.Coin.String(), context.Coins().GetCoin(data.Coin).GetFullSymbol(), stake.String(), data.Value.String())),
			}
		}
	}

	value := big.NewInt(0).Set(data.Value)
	if waitList := context.WaitList().Get(sender, data.To, data.Coin); waitList != nil {
		value.Add(value, waitList.Value)
	}

	if !context.Candidates().IsDelegatorStakeSufficient(sender, data.To, data.Coin, value) {
		coin := context.Coins().GetCoin(data.Coin)
		return &Response{
			Code: code.TooLowStake,
			Log:  "Stake is too low",
			Info: EncodeError(code.NewTooLowStake(sender.String(), data.To.String(), value.String(), data.Coin.String(), coin.GetFullSymbol())),
		}
	}

	return nil
}

func (data MoveStakeData) String() string {
	return fmt.Sprintf("MOVE STAKE")
}

func (data MoveStakeData) Gas() int64 {
	return commissions.MoveStakeData
}

func (data MoveStakeData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		// todo: logic

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeMoveStake)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
