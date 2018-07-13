package transaction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type SetCandidateOnData struct {
	PubKey []byte
}

func (data SetCandidateOnData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pubkey"`
	}{
		PubKey: fmt.Sprintf("Mp%x", data.PubKey),
	})
}

func (data SetCandidateOnData) String() string {
	return fmt.Sprintf("SET CANDIDATE ONLINE pubkey: %x",
		data.PubKey)
}

func (data SetCandidateOnData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

func (data SetCandidateOnData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commission.Mul(commission, CommissionMultiplier)

	if context.GetBalance(sender, types.GetBaseCoin()).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %data. Wanted %d ", sender.String(), commission)}
	}

	if !context.CandidateExists(data.PubKey) {
		return Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key (%x) not found", data.PubKey)}
	}

	candidate := context.GetStateCandidate(data.PubKey)

	if bytes.Compare(candidate.CandidateAddress.Bytes(), sender.Bytes()) != 0 {
		return Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commission)

		context.SubBalance(sender, types.GetBaseCoin(), commission)
		context.SetCandidateOnline(data.PubKey)
		context.SetNonce(sender, tx.Nonce)
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}

type SetCandidateOffData struct {
	PubKey []byte
}

func (data SetCandidateOffData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pubkey"`
	}{
		PubKey: fmt.Sprintf("Mp%x", data.PubKey),
	})
}

func (data SetCandidateOffData) String() string {
	return fmt.Sprintf("SET CANDIDATE OFFLINE pubkey: %x",
		data.PubKey)
}

func (data SetCandidateOffData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

func (data SetCandidateOffData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commission.Mul(commission, CommissionMultiplier)

	if context.GetBalance(sender, types.GetBaseCoin()).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %data. Wanted %d ", sender.String(), commission)}
	}

	if !context.CandidateExists(data.PubKey) {
		return Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key not found")}
	}

	candidate := context.GetStateCandidate(data.PubKey)

	if bytes.Compare(candidate.CandidateAddress.Bytes(), sender.Bytes()) != 0 {
		return Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commission)

		context.SubBalance(sender, types.GetBaseCoin(), commission)
		context.SetCandidateOffline(data.PubKey)
		context.SetNonce(sender, tx.Nonce)
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
