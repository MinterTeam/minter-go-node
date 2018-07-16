package transaction

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"math/big"
)

const unbondPeriod = 518400

type UnbondData struct {
	PubKey []byte
	Coin   types.CoinSymbol
	Value  *big.Int
}

func (data UnbondData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string           `json:"pub_key"`
		Coin   types.CoinSymbol `json:"coin"`
		Value  string           `json:"value"`
	}{
		PubKey: fmt.Sprintf("Mp%x", data.PubKey),
		Coin:   data.Coin,
		Value:  data.Value.String(),
	})
}

func (data UnbondData) String() string {
	return fmt.Sprintf("UNBOND pubkey:%s",
		hexutil.Encode(data.PubKey[:]))
}

func (data UnbondData) Gas() int64 {
	return commissions.UnbondTx
}

func (data UnbondData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	commission := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commission.Mul(commission, CommissionMultiplier)

	if context.GetBalance(sender, types.GetBaseCoin()).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), commission)}
	}

	if !context.CandidateExists(data.PubKey) {
		return Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key not found")}
	}

	candidate := context.GetStateCandidate(data.PubKey)

	stake := candidate.GetStakeOfAddress(sender, data.Coin)

	if stake == nil {
		return Response{
			Code: code.StakeNotFound,
			Log:  fmt.Sprintf("Stake of current user not found")}
	}

	if stake.Value.Cmp(data.Value) < 0 {
		return Response{
			Code: code.InsufficientStake,
			Log:  fmt.Sprintf("Insufficient stake for sender account")}
	}

	if !isCheck {
		// now + 31 days
		unbondAtBlock := currentBlock + unbondPeriod

		rewardPull.Add(rewardPull, commission)

		context.SubBalance(sender, types.GetBaseCoin(), commission)
		context.SubStake(sender, data.PubKey, data.Coin, data.Value)
		context.GetOrNewStateFrozenFunds(unbondAtBlock).AddFund(sender, data.PubKey, data.Coin, data.Value)
		context.SetNonce(sender, tx.Nonce)
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
