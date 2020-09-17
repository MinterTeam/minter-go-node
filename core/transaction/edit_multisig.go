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

type EditMultisigData struct {
	Threshold uint
	Weights   []uint
	Addresses []types.Address
}

func (data EditMultisigData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	sender, _ := tx.Sender()

	if !context.Accounts().GetAccount(sender).IsMultisig() {
		return &Response{
			Code: code.MultisigNotExists,
			Log:  "Multisig does not exists",
			Info: EncodeError(code.NewMultisigNotExists(sender.String())),
		}
	}

	lenWeights := len(data.Weights)
	if lenWeights > 32 {
		return &Response{
			Code: code.TooLargeOwnersList,
			Log:  fmt.Sprintf("Owners list is limited to 32 items"),
			Info: EncodeError(code.NewTooLargeOwnersList(strconv.Itoa(lenWeights), "32")),
		}
	}

	lenAddresses := len(data.Addresses)
	if lenAddresses != lenWeights {
		return &Response{
			Code: code.DifferentCountAddressesAndWeights,
			Log:  fmt.Sprintf("Different count addresses and weights"),
			Info: EncodeError(code.NewDifferentCountAddressesAndWeights(fmt.Sprintf("%d", lenAddresses), fmt.Sprintf("%d", lenWeights))),
		}
	}

	for i, weight := range data.Weights {
		if weight > 1023 {
			return &Response{
				Code: code.IncorrectWeights,
				Log:  "Incorrect multisig weights",
				Info: EncodeError(code.NewIncorrectWeights(data.Addresses[i].String(), strconv.Itoa(int(weight)), "1024")),
			}
		}
	}

	usedAddresses := map[types.Address]bool{}
	for _, address := range data.Addresses {
		if usedAddresses[address] {
			return &Response{
				Code: code.DuplicatedAddresses,
				Log:  fmt.Sprintf("Duplicated multisig addresses"),
				Info: EncodeError(code.NewDuplicatedAddresses(address.String())),
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
			Code: code.IncorrectTotalWeights,
			Log:  "Incorrect multisig weights",
			Info: EncodeError(code.NewIncorrectTotalWeights(fmt.Sprintf("%d", totalWeight), fmt.Sprintf("%d", data.Threshold))),
		}
	}

	return nil
}

func (data EditMultisigData) String() string {
	return "EDIT MULTISIG OWNERS"
}

func (data EditMultisigData) Gas() int64 {
	return commissions.EditMultisigData
}

func (data EditMultisigData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
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
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditMultisig)}))},
		kv.Pair{Key: []byte("tx.from"), Value: address},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
