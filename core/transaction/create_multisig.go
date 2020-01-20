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
	"github.com/tendermint/tendermint/libs/kv"
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
	lenWeights := len(data.Weights)
	if lenWeights > 32 {
		return &Response{
			Code: code.TooLargeOwnersList,
			Log:  fmt.Sprintf("Owners list is limited to 32 items")}
	}

	lenAddresses := len(data.Addresses)
	if lenAddresses != lenWeights {
		return &Response{
			Code: code.IncorrectWeights,
			Log:  fmt.Sprintf("Incorrect multisig weights"),
			Info: EncodeError(map[string]string{
				"count_weights":   fmt.Sprintf("%d", lenWeights),
				"count_addresses": fmt.Sprintf("%d", lenAddresses),
			}),
		}
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
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.Reserve().String(), commissionInBaseCoin.String()),
				Info: EncodeError(map[string]string{
					"has_reserve": coin.Reserve().String(),
					"commission":  commissionInBaseCoin.String(),
					"gas_coin":    coin.CName,
				}),
			}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	if context.Accounts.GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin),
			Info: EncodeError(map[string]string{
				"sender":       sender.String(),
				"needed_value": commission.String(),
				"gas_coin":     fmt.Sprintf("%s", tx.GasCoin),
			}),
		}
	}

	msigAddress := (&accounts.Multisig{
		Weights:   data.Weights,
		Threshold: data.Threshold,
		Addresses: data.Addresses,
	}).Address()

	if context.Accounts.Exists(msigAddress) {
		return Response{
			Code: code.MultisigExists,
			Log:  fmt.Sprintf("Multisig %s already exists", msigAddress.String()),
			Info: EncodeError(map[string]string{
				"multisig_address": msigAddress.String(),
			}),
		}
	}

	if !isCheck {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		context.Coins.SubVolume(tx.GasCoin, commission)
		context.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		context.Accounts.SubBalance(sender, tx.GasCoin, commission)
		context.Accounts.SetNonce(sender, tx.Nonce)

		context.Accounts.CreateMultisig(data.Weights, data.Addresses, data.Threshold)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeCreateMultisig)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.created_multisig"), Value: []byte(hex.EncodeToString(msigAddress[:]))},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
