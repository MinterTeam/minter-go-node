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

type DelegateData struct {
	PubKey []byte
	Coin   types.CoinSymbol
	Stake  *big.Int
}

func (data DelegateData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string           `json:"pub_key"`
		Coin   types.CoinSymbol `json:"coin"`
		Stake  string           `json:"stake"`
	}{
		PubKey: fmt.Sprintf("Mp%x", data.PubKey),
		Coin:   data.Coin,
		Stake:  data.Stake.String(),
	})
}

func (data DelegateData) String() string {
	return fmt.Sprintf("DELEGATE ubkey:%s ",
		hexutil.Encode(data.PubKey[:]))
}

func (data DelegateData) Gas() int64 {
	return commissions.DelegateTx
}

func (data DelegateData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if data.Coin != types.GetBaseCoin() {
		coin := context.GetStateCoin(data.Coin)

		if coin.ReserveBalance().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.ReserveBalance().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	totalTxCost := big.NewInt(0).Add(data.Stake, commission)

	if context.GetBalance(sender, data.Coin).Cmp(totalTxCost) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %d ", sender.String(), totalTxCost)}
	}

	if !context.CandidateExists(data.PubKey) {
		return Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key not found")}
	}

	if !isCheck {
		rewardPull.Add(rewardPull, commission)

		context.SubBalance(sender, data.Coin, totalTxCost)
		context.Delegate(sender, data.PubKey, data.Coin, data.Stake)
		context.SetNonce(sender, tx.Nonce)
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
