package transaction

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type EditMultisigData struct {
	Threshold uint32
	Weights   []uint32
	Addresses []types.Address
}

func (data EditMultisigData) Gas() int64 {
	return gasEditMultisig
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

func (data EditMultisigData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := price
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

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  []*swap.OrderDetail
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.CommissionCoin(), types.GetBaseCoinID(), commission, big.NewInt(0))
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  ownersCom,
			}
			for _, value := range ownersCom {
				deliverState.Accounts.AddBalance(value.Owner, tx.CommissionCoin(), value.ValueBigInt)
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.CommissionCoin(), commission)
			deliverState.Coins.SubReserve(tx.CommissionCoin(), commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		deliverState.Accounts.EditMultisig(data.Threshold, data.Weights, data.Addresses, sender)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
