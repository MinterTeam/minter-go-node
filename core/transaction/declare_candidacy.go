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

const minCommission = 0
const maxCommission = 100

type DeclareCandidacyData struct {
	Address    types.Address
	PubKey     []byte
	Commission uint
	Coin       types.CoinSymbol
	Stake      *big.Int
}

func (data DeclareCandidacyData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address    types.Address    `json:"address"`
		PubKey     string           `json:"pub_key"`
		Commission uint             `json:"commission"`
		Coin       types.CoinSymbol `json:"coin"`
		Stake      string           `json:"stake"`
	}{
		Address:    data.Address,
		PubKey:     fmt.Sprintf("Mp%x", data.PubKey),
		Commission: data.Commission,
		Coin:       data.Coin,
		Stake:      data.Stake.String(),
	})
}

func (data DeclareCandidacyData) String() string {
	return fmt.Sprintf("DECLARE CANDIDACY address:%s pubkey:%s commission: %d ",
		data.Address.String(), hexutil.Encode(data.PubKey[:]), data.Commission)
}

func (data DeclareCandidacyData) Gas() int64 {
	return commissions.DeclareCandidacyTx
}

func (data DeclareCandidacyData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	if len(data.PubKey) != 32 {
		return Response{
			Code: code.IncorrectPubKey,
			Log:  fmt.Sprintf("Incorrect PubKey")}
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

	if context.CandidateExists(data.PubKey) {
		return Response{
			Code: code.CandidateExists,
			Log:  fmt.Sprintf("Candidate with such public key (%x) already exists", data.PubKey)}
	}

	if data.Commission < minCommission || data.Commission > maxCommission {
		return Response{
			Code: code.WrongCommission,
			Log:  fmt.Sprintf("Commission should be between 0 and 100")}
	}

	// TODO: limit number of candidates to prevent flooding

	if !isCheck {
		rewardPull.Add(rewardPull, commissionInBaseCoin)

		context.SubBalance(sender, data.Coin, data.Stake)
		context.SubBalance(sender, tx.GasCoin, commission)
		context.CreateCandidate(data.Address, data.PubKey, data.Commission, uint(currentBlock), data.Coin, data.Stake)
		context.SetNonce(sender, tx.Nonce)
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
