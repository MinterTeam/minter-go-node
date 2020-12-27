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
	"strconv"
)

type EditCommissionData struct {
	PubKey     types.Pubkey
	Commission uint32
}

func (data EditCommissionData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCommissionData) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	errResp := checkCandidateOwnership(data, tx, context)
	if errResp != nil {
		return errResp
	}

	candidate := context.Candidates().GetCandidate(data.PubKey)

	maxNewCommission, minNewCommission := candidate.Commission+10, candidate.Commission-10
	if maxNewCommission > maxCommission {
		maxNewCommission = maxCommission
	}
	if minNewCommission < minCommission {
		minNewCommission = minCommission
	}
	if data.Commission < minNewCommission || data.Commission > maxNewCommission {
		return &Response{
			Code: code.WrongCommission,
			Log:  fmt.Sprintf("You want change commission from %d to %d, but you can change no more than 10 units, because commission should be between %d and %d", candidate.Commission, data.Commission, minNewCommission, maxNewCommission),
			Info: EncodeError(code.NewWrongCommission(fmt.Sprintf("%d", data.Commission), strconv.Itoa(int(minNewCommission)), strconv.Itoa(int(maxNewCommission)))),
		}
	}

	if candidate.LastEditCommissionHeight+3*types.GetUnbondPeriod() > block {
		return &Response{
			Code: code.PeriodLimitReached,
			Log:  fmt.Sprintf("You cannot change the commission more than once every %d blocks, the last change was on block %d", 3*types.GetUnbondPeriod(), candidate.LastEditCommissionHeight),
			Info: EncodeError(code.NewPeriodLimitReached(strconv.Itoa(int(candidate.LastEditCommissionHeight+3*types.GetUnbondPeriod())), strconv.Itoa(int(candidate.LastEditCommissionHeight)))),
		}
	}

	return nil
}

func (data EditCommissionData) String() string {
	return fmt.Sprintf("EDIT COMMISSION: %s", data.PubKey)
}

func (data EditCommissionData) Gas() int64 {
	return commissions.EditCommissionData
}

func (data EditCommissionData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState, currentBlock)
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

		deliverState.Candidates.EditCommission(data.PubKey, data.Commission, currentBlock)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditCommission)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
