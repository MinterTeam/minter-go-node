package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
	"strconv"
)

type CreateMultisigData struct {
	Threshold uint
	Weights   []uint
	Addresses []types.Address
}

func (data CreateMultisigData) MarshalJSON() ([]byte, error) {
	var weights []string
	for _, weight := range data.Weights {
		weights = append(weights, strconv.Itoa(int(weight)))
	}

	return json.Marshal(struct {
		Threshold string          `json:"threshold"`
		Weights   []string        `json:"weights"`
		Addresses []types.Address `json:"addresses"`
	}{
		Threshold: strconv.Itoa(int(data.Threshold)),
		Weights:   weights,
		Addresses: data.Addresses,
	})
}

func (data CreateMultisigData) TotalSpend(tx *Transaction, context *state.State) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data CreateMultisigData) BasicCheck(tx *Transaction, context *state.State) *Response {
	if len(data.Weights) > 32 {
		return &Response{
			Code: code.TooLargeOwnersList,
			Log:  fmt.Sprintf("Owners list is limited to 32 items")}
	}

	if len(data.Addresses) != len(data.Weights) {
		return &Response{
			Code: code.IncorrectWeights,
			Log:  fmt.Sprintf("Incorrect multisig weights")}
	}

	for _, weight := range data.Weights {
		if weight > 1023 {
			return &Response{
				Code: code.IncorrectWeights,
				Log:  fmt.Sprintf("Incorrect multisig weights")}
		}
	}

	return nil
}

func (data CreateMultisigData) String() string {
	return fmt.Sprintf("CREATE MULTISIG")
}

func (data CreateMultisigData) Gas() int64 {
	return commissions.CreateMultisig
}

func (data CreateMultisigData) Run(tx *Transaction, context *state.State, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	response := data.BasicCheck(tx, context)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := context.Coins.GetCoin(tx.GasCoin)

		errResp := CheckReserveUnderflow(coin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.Reserve().String(), commissionInBaseCoin.String())}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	if context.Accounts.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin)}
	}

	msigAddress := (&accounts.Multisig{
		Weights:   data.Weights,
		Threshold: data.Threshold,
		Addresses: data.Addresses,
	}).Address()

	if context.Accounts.Exists(msigAddress) {
		return Response{
			Code: code.MultisigExists,
			Log:  fmt.Sprintf("Multisig %s already exists", msigAddress.String())}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.Coins.SubVolume(tx.GasCoin, commission)
		context.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		context.Accounts.SubBalance(sender, tx.GasCoin, commission)
		context.Accounts.SetNonce(sender, tx.Nonce)

		context.Accounts.CreateMultisig(data.Weights, data.Addresses, data.Threshold)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeCreateMultisig)}))},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.created_multisig"), Value: []byte(hex.EncodeToString(msigAddress[:]))},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
