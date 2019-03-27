package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
)

const minCommission = 0
const maxCommission = 100

type DeclareCandidacyData struct {
	Address    types.Address    `json:"address"`
	PubKey     types.Pubkey     `json:"pub_key"`
	Commission uint             `json:"commission"`
	Coin       types.CoinSymbol `json:"coin"`
	Stake      *big.Int         `json:"stake"`
}

func (data DeclareCandidacyData) TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data DeclareCandidacyData) BasicCheck(tx *Transaction, context *state.StateDB) *Response {
	if data.PubKey == nil || data.Stake == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	if !context.CoinExists(tx.GasCoin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if len(data.PubKey) != 32 {
		return &Response{
			Code: code.IncorrectPubKey,
			Log:  fmt.Sprintf("Incorrect PubKey")}
	}

	if context.CandidateExists(data.PubKey) {
		return &Response{
			Code: code.CandidateExists,
			Log:  fmt.Sprintf("Candidate with such public key (%s) already exists", data.PubKey.String())}
	}

	if data.Commission < minCommission || data.Commission > maxCommission {
		return &Response{
			Code: code.WrongCommission,
			Log:  fmt.Sprintf("Commission should be between 0 and 100")}
	}

	return nil
}

func (data DeclareCandidacyData) String() string {
	return fmt.Sprintf("DECLARE CANDIDACY address:%s pubkey:%s commission: %d",
		data.Address.String(), data.PubKey.String(), data.Commission)
}

func (data DeclareCandidacyData) Gas() int64 {
	return commissions.DeclareCandidacyTx
}

func (data DeclareCandidacyData) Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	maxCandidatesCount := validators.GetCandidatesCountForBlock(currentBlock)

	if context.CandidatesCount() >= maxCandidatesCount && !context.IsNewCandidateStakeSufficient(data.Coin, data.Stake) {
		return Response{
			Code: code.TooLowStake,
			Log:  fmt.Sprintf("Given stake is too low")}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
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

	if context.GetBalance(sender, data.Coin).Cmp(data.Stake) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.Stake, data.Coin)}
	}

	if context.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin)}
	}

	if data.Coin == tx.GasCoin {
		totalTxCost := big.NewInt(0)
		totalTxCost.Add(totalTxCost, data.Stake)
		totalTxCost.Add(totalTxCost, commission)

		if context.GetBalance(sender, tx.GasCoin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), tx.GasCoin)}
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.SubCoinReserve(tx.GasCoin, commissionInBaseCoin)
		context.SubCoinVolume(tx.GasCoin, commission)

		context.SubBalance(sender, data.Coin, data.Stake)
		context.SubBalance(sender, tx.GasCoin, commission)
		context.CreateCandidate(data.Address, sender, data.PubKey, data.Commission, uint(currentBlock), data.Coin, data.Stake)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeDeclareCandidacy)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
