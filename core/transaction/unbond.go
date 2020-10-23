package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

const unbondPeriod = 518400

type UnbondData struct {
	PubKey types.Pubkey
	Coin   types.CoinID
	Value  *big.Int
}

func (data UnbondData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Value == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if !context.Coins().Exists(data.Coin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Coin),
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !context.Candidates().Exists(data.PubKey) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  "Candidate with such public key not found",
			Info: EncodeError(code.NewCandidateNotFound(data.PubKey.String())),
		}
	}

	sender, _ := tx.Sender()

	if waitlist := context.WaitList().Get(sender, data.PubKey, data.Coin); waitlist != nil {
		value := big.NewInt(0).Sub(data.Value, waitlist.Value)
		if value.Sign() < 1 {
			return nil
		}
		return &Response{
			Code: code.InsufficientWaitList,
			Log:  "Insufficient amount at waitlist for sender account",
			Info: EncodeError(code.NewInsufficientWaitList(waitlist.Value.String(), data.Value.String())),
		}
	}

	stake := context.Candidates().GetStakeValueOfAddress(data.PubKey, sender, data.Coin)

	if stake == nil {
		return &Response{
			Code: code.StakeNotFound,
			Log:  "Stake of current user not found",
			Info: EncodeError(code.NewStakeNotFound(data.PubKey.String(), sender.String(), data.Coin.String(), context.Coins().GetCoin(data.Coin).GetFullSymbol())),
		}
	}

	if stake.Cmp(data.Value) < 0 {
		return &Response{
			Code: code.InsufficientStake,
			Log:  "Insufficient stake for sender account",
			Info: EncodeError(code.NewInsufficientStake(data.PubKey.String(), sender.String(), data.Coin.String(), context.Coins().GetCoin(data.Coin).GetFullSymbol(), stake.String(), data.Value.String())),
		}
	}

	return nil
}

func (data UnbondData) String() string {
	return fmt.Sprintf("UNBOND pubkey:%s",
		hexutil.Encode(data.PubKey[:]))
}

func (data UnbondData) Gas() int64 {
	return commissions.UnbondTx
}

func (data UnbondData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.BasicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

	if !tx.GasCoin.IsBaseCoin() {
		errResp := CheckReserveUnderflow(gasCoin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		// now + 30 days
		unbondAtBlock := currentBlock + unbondPeriod

		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		deliverState.Coins.SubVolume(tx.GasCoin, commission)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)

		if waitList := deliverState.Waitlist.Get(sender, data.PubKey, data.Coin); waitList != nil {
			diffValue := big.NewInt(0).Sub(data.Value, waitList.Value)
			deliverState.Waitlist.Delete(sender, data.PubKey, data.Coin)
			if diffValue.Sign() == -1 {
				deliverState.Waitlist.AddWaitList(sender, data.PubKey, data.Coin, big.NewInt(0).Neg(diffValue))
			}
		} else {
			deliverState.Candidates.SubStake(sender, data.PubKey, data.Coin, data.Value)
		}

		deliverState.FrozenFunds.AddFund(unbondAtBlock, sender, data.PubKey, deliverState.Candidates.ID(data.PubKey), data.Coin, data.Value)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeUnbond)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
