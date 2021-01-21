package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/commission"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
)

type EditMultisigData struct {
	Threshold uint32
	Weights   []uint32
	Addresses []types.Address
}

func (data EditMultisigData) TxType() TxType {
	return TypeEditMultisig
}

func (data EditMultisigData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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
			Log:  "Owners list is limited to 32 items",
			Info: EncodeError(code.NewTooLargeOwnersList(strconv.Itoa(lenWeights), "32")),
		}
	}

	lenAddresses := len(data.Addresses)
	if lenAddresses != lenWeights {
		return &Response{
			Code: code.DifferentCountAddressesAndWeights,
			Log:  "Different count addresses and weights",
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
				Log:  "Duplicated multisig addresses",
				Info: EncodeError(code.NewDuplicatedAddresses(address.String())),
			}
		}

		usedAddresses[address] = true
	}

	var totalWeight uint32
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

func (data EditMultisigData) CommissionData(price *commission.Price) *big.Int {
	return price.EditMultisig
}

func (data EditMultisigData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int, gas int64) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		deliverState.Accounts.EditMultisig(data.Threshold, data.Weights, data.Addresses, sender)
	}

	address := []byte(hex.EncodeToString(sender[:]))
	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.gas"), Value: []byte(strconv.Itoa(int(gas)))},
		kv.Pair{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditMultisig)}))},
		kv.Pair{Key: []byte("tx.from"), Value: address},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   gas,
		GasWanted: gas,
		Tags:      tags,
	}
}
