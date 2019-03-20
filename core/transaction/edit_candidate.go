package transaction

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
)

type CandidateTx interface {
	GetPubKey() types.Pubkey
}

type EditCandidateData struct {
	PubKey        types.Pubkey  `json:"pub_key"`
	RewardAddress types.Address `json:"reward_address"`
	OwnerAddress  types.Address `json:"owner_address"`
}

func (data EditCandidateData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCandidateData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data EditCandidateData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data EditCandidateData) String() string {
	return fmt.Sprintf("EDIT CANDIDATE pubkey: %x",
		data.PubKey)
}

func (data EditCandidateData) Gas() int64 {
	return commissions.EditCandidate
}

func (data EditCandidateData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
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

		context.SubCoinReserve(tx.GasCoin, commissionInBaseCoin)
		context.SubCoinVolume(tx.GasCoin, commission)

		context.SubBalance(sender, tx.GasCoin, commission)
		context.EditCandidate(data.PubKey, data.RewardAddress, data.OwnerAddress)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditCandidate)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}

func checkCandidateOwnership(data CandidateTx, tx *Transaction, context *state.StateDB) *Response {
	if data.GetPubKey() == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if !context.CandidateExists(data.GetPubKey()) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key (%s) not found", data.GetPubKey().String())}
	}

	candidate := context.GetStateCandidate(data.GetPubKey())
	sender, _ := tx.Sender()
	if !bytes.Equal(candidate.OwnerAddress.Bytes(), sender.Bytes()) {
		return &Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	return nil
}
