package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
)

type DelegateData struct {
	PubKey types.Pubkey     `json:"pub_key"`
	Coin   types.CoinSymbol `json:"coin"`
	Value  *big.Int         `json:"value"`
}

func (data DelegateData) TotalSpend(tx *Transaction, context *state.State) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data DelegateData) BasicCheck(tx *Transaction, context *state.State) *Response {
	if data.Value == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if !context.Coins.Exists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if data.Value.Cmp(types.Big0) < 1 {
		return &Response{
			Code: code.StakeShouldBePositive,
			Log:  fmt.Sprintf("Stake should be positive")}
	}

	candidate := context.Candidates.GetCandidate(data.PubKey)
	if candidate == nil {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key not found")}
	}

	sender, _ := tx.Sender()
	if context.Candidates.StakesCount(candidate.PubKey) >= candidates.MaxDelegatorsPerCandidate && !context.Candidates.IsDelegatorStakeSufficient(sender, data.PubKey, data.Coin, data.Value) {
		return &Response{
			Code: code.TooLowStake,
			Log:  fmt.Sprintf("Stake is too low")}
	}

	return nil
}

func (data DelegateData) String() string {
	return fmt.Sprintf("DELEGATE pubkey:%s ",
		hexutil.Encode(data.PubKey[:]))
}

func (data DelegateData) Gas() int64 {
	return commissions.DelegateTx
}

func (data DelegateData) Run(tx *Transaction, context *state.State, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.Coins.GetCoin(tx.GasCoin)

		if coin.ReserveBalance.Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance.String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume, coin.ReserveBalance, coin.Crr, commissionInBaseCoin)
	}

	if context.Accounts.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin)}
	}

	if context.Accounts.GetBalance(sender, data.Coin).Cmp(data.Value) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.Value, data.Coin)}
	}

	if data.Coin == tx.GasCoin {
		totalTxCost := big.NewInt(0)
		totalTxCost.Add(totalTxCost, data.Value)
		totalTxCost.Add(totalTxCost, commission)

		if context.Accounts.GetBalance(sender, tx.GasCoin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), tx.GasCoin)}
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		context.Coins.SubVolume(tx.GasCoin, commission)

		context.Accounts.SubBalance(sender, tx.GasCoin, commission)
		context.Accounts.SubBalance(sender, data.Coin, data.Value)
		context.Candidates.Delegate(sender, data.PubKey, data.Coin, data.Value)
		context.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeDelegate)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
