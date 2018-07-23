package transaction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

type SetCandidateOnData struct {
	PubKey []byte
}

func (data SetCandidateOnData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pub_key"`
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
			Log:  fmt.Sprintf("Candidate with such public key (%x) not found", data.PubKey)}
	}

	candidate := context.GetStateCandidate(data.PubKey)

	if !bytes.Equal(candidate.CandidateAddress.Bytes(), sender.Bytes()) {
		return Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commissionInBaseCoin)

		context.SubBalance(sender, tx.GasCoin, commission)
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
		PubKey string `json:"pub_key"`
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
			Log:  fmt.Sprintf("Candidate with such public key (%x) not found", data.PubKey)}
	}

	candidate := context.GetStateCandidate(data.PubKey)

	if !bytes.Equal(candidate.CandidateAddress.Bytes(), sender.Bytes()) {
		return Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commissionInBaseCoin)

		context.SubBalance(sender, tx.GasCoin, commission)
		context.SetCandidateOffline(data.PubKey)
		context.SetNonce(sender, tx.Nonce)
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
