package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
)

type EditMultisigOwnersData struct {
	Threshold uint
	Weights   []uint
	Addresses []types.Address
}

func (data EditMultisigOwnersData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	sender, _ := tx.Sender()

	if !context.Accounts().GetAccount(sender).IsMultisig() {
		return &Response{
			Code: code.MultisigNotExists,
			Log:  "Multisig does not exists",
			Info: EncodeError(map[string]string{
				"code":             strconv.Itoa(int(code.MultisigNotExists)),
				"multisig_address": sender.String(),
			}),
		}
	}

	lenWeights := len(data.Weights)
	if lenWeights > 32 {
		return &Response{
			Code: code.TooLargeOwnersList,
			Log:  fmt.Sprintf("Owners list is limited to 32 items"),
			Info: EncodeError(map[string]string{
				"code": strconv.Itoa(int(code.TooLargeOwnersList)),
			}),
		}
	}

	lenAddresses := len(data.Addresses)
	if lenAddresses != lenWeights {
		return &Response{
			Code: code.IncorrectWeights,
			Log:  fmt.Sprintf("Incorrect multisig weights"),
			Info: EncodeError(map[string]string{
				"code":            strconv.Itoa(int(code.IncorrectWeights)),
				"count_weights":   fmt.Sprintf("%d", lenWeights),
				"count_addresses": fmt.Sprintf("%d", lenAddresses),
			}),
		}
	}

	for _, weight := range data.Weights {
		if weight > 1023 {
			return &Response{
				Code: code.IncorrectWeights,
				Log:  "Incorrect multisig weights",
				Info: EncodeError(map[string]string{
					"code": strconv.Itoa(int(code.IncorrectWeights)),
				}),
			}
		}
	}

	usedAddresses := map[types.Address]bool{}
	for _, address := range data.Addresses {
		if usedAddresses[address] {
			return &Response{
				Code: code.DuplicatedAddresses,
				Log:  fmt.Sprintf("Duplicated multisig addresses"),
				Info: EncodeError(map[string]string{
					"code": strconv.Itoa(int(code.DuplicatedAddresses)),
				}),
			}
		}

		usedAddresses[address] = true
	}

	var totalWeight uint
	for _, weight := range data.Weights {
		totalWeight += weight
	}
	if data.Threshold > totalWeight {
		return &Response{
			Code: code.IncorrectWeights,
			Log:  "Incorrect multisig weights",
			Info: EncodeError(map[string]string{
				"code":         strconv.Itoa(int(code.IncorrectWeights)),
				"total_weight": fmt.Sprintf("%d", totalWeight),
				"threshold":    fmt.Sprintf("%d", data.Threshold),
			}),
		}
	}

	return nil
}

func (data EditMultisigOwnersData) String() string {
	return "EDIT MULTISIG OWNERS"
}

func (data EditMultisigOwnersData) Gas() int64 {
	return commissions.EditMultisigOwnersData
}

func (data EditMultisigOwnersData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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
					"code":           strconv.Itoa(int(code.CoinReserveNotSufficient)),
					"has_reserve":    gasCoin.Reserve().String(),
					"required_value": commissionInBaseCoin.String(),
					"coin_symbol":    gasCoin.GetFullSymbol(),
				}),
			}
		}

		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(map[string]string{
				"code":         strconv.Itoa(int(code.InsufficientFunds)),
				"sender":       sender.String(),
				"needed_value": commission.String(),
				"coin_symbol":  gasCoin.GetFullSymbol(),
			}),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(tx.GasCoin, commission)
		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		deliverState.Accounts.EditMultisig(data.Threshold, data.Weights, data.Addresses, sender)
	}

	address := []byte(hex.EncodeToString(sender[:]))
	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditMultisigOwner)}))},
		kv.Pair{Key: []byte("tx.from"), Value: address},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
