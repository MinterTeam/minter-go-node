package transaction

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"math/big"
)

type SetCandidateOnData struct {
	PubKey types.Pubkey `json:"pub_key"`
}

func (data SetCandidateOnData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data SetCandidateOnData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if !context.CandidateExists(data.PubKey) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key (%s) not found", data.PubKey.String())}
	}

	candidate := context.GetStateCandidate(data.PubKey)
	sender, _ := tx.Sender()
	if !bytes.Equal(candidate.CandidateAddress.Bytes(), sender.Bytes()) {
		return &Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	return nil
}

func (data SetCandidateOnData) String() string {
	return fmt.Sprintf("SET CANDIDATE ONLINE pubkey: %x",
		data.PubKey)
}

func (data SetCandidateOnData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

func (data SetCandidateOnData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
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

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

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
	PubKey types.Pubkey `json:"pub_key"`
}

func (data SetCandidateOffData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data SetCandidateOffData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if !context.CandidateExists(data.PubKey) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key (%s) not found", data.PubKey.String())}
	}

	candidate := context.GetStateCandidate(data.PubKey)
	sender, _ := tx.Sender()
	if !bytes.Equal(candidate.CandidateAddress.Bytes(), sender.Bytes()) {
		return &Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	return nil
}

func (data SetCandidateOffData) String() string {
	return fmt.Sprintf("SET CANDIDATE OFFLINE pubkey: %x",
		data.PubKey)
}

func (data SetCandidateOffData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

func (data SetCandidateOffData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response {
	sender, _ := tx.Sender()

	if currentBlock > 7000 { // TODO: remove
		response := data.BasicCheck(tx, context)
		if response != nil {
			return *response
		}
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
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

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

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
