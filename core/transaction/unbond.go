package transaction

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
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

	if !context.CoinExists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if tx.GasCoin != types.GetBaseCoin() {
		coin := context.GetStateCoin(tx.GasCoin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	if context.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin)}
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

		rewardPull.Add(rewardPull, commissionInBaseCoin)

		context.SubBalance(sender, tx.GasCoin, commission)
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
