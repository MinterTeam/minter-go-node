package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type CandidateTx interface {
	GetPubKey() types.Pubkey
}

type EditCandidateData struct {
	PubKey         types.Pubkey
	RewardAddress  types.Address
	OwnerAddress   types.Address
	ControlAddress types.Address
}

func (data EditCandidateData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCandidateData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data EditCandidateData) String() string {
	return fmt.Sprintf("EDIT CANDIDATE pubkey: %x",
		data.PubKey)
}

func (data EditCandidateData) Gas() int64 {
	return commissions.EditCandidate
}

func (data EditCandidateData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		deliverState.Coins.SubVolume(tx.GasCoin, commission)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Candidates.Edit(data.PubKey, data.RewardAddress, data.OwnerAddress, data.ControlAddress)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditCandidate)}))},
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

func checkCandidateOwnership(data CandidateTx, tx *Transaction, context *state.CheckState) *Response {
	if !context.Candidates().Exists(data.GetPubKey()) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key (%s) not found", data.GetPubKey().String()),
			Info: EncodeError(code.NewCandidateNotFound(data.GetPubKey().String())),
		}
	}

	owner := context.Candidates().GetCandidateOwner(data.GetPubKey())
	sender, _ := tx.Sender()
	if owner != sender {
		return &Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  "Sender is not an owner of a candidate",
			Info: EncodeError(code.NewIsNotOwnerOfCandidate(sender.String(), data.GetPubKey().String(), owner.String(), "")),
		}
	}

	return nil
}
