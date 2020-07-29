package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type CreateMultisigData struct {
	Threshold uint
	Weights   []uint
	Addresses []types.Address
}

func (data CreateMultisigData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data CreateMultisigData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
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

	usedAddresses := map[types.Address]bool{}
	for _, address := range data.Addresses {
		if usedAddresses[address] {
			return &Response{
				Code: code.DuplicatedAddresses,
				Log:  fmt.Sprintf("Duplicated multisig addresses")}
		}

		usedAddresses[address] = true
	}

	return nil
}

func (data CreateMultisigData) String() string {
	return fmt.Sprintf("CREATE MULTISIG")
}

func (data CreateMultisigData) Gas() int64 {
	return commissions.CreateMultisig
}

func (data CreateMultisigData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.BasicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

	if !tx.GasCoin.IsBaseCoin() {
		errResp := CheckReserveUnderflow(gasCoin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		if gasCoin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", gasCoin.Reserve().String(), commissionInBaseCoin.String()),
				Info: EncodeError(map[string]string{
					"has_reserve": gasCoin.Reserve().String(),
					"commission":  commissionInBaseCoin.String(),
					"gas_coin":    gasCoin.GetFullSymbol(),
				}),
			}
		}

		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, gasCoin.GetFullSymbol()),
			Info: EncodeError(map[string]string{
				"sender":       sender.String(),
				"needed_value": commission.String(),
				"gas_coin":     gasCoin.GetFullSymbol(),
			}),
		}
	}

	msigAddress := (&accounts.Multisig{
		Weights:   data.Weights,
		Threshold: data.Threshold,
		Addresses: data.Addresses,
	}).Address()

	if checkState.Accounts().ExistsMultisig(msigAddress) {
		return Response{
			Code: code.MultisigExists,
			Log:  fmt.Sprintf("Multisig %s already exists", msigAddress.String()),
			Info: EncodeError(map[string]string{
				"multisig_address": msigAddress.String(),
			}),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(tx.GasCoin, commission)
		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		deliverState.Accounts.CreateMultisig(data.Weights, data.Addresses, data.Threshold, currentBlock)
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
