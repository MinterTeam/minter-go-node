package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/accounts"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type CreateMultisigData struct {
	Threshold uint32
	Weights   []uint32
	Addresses []types.Address
}

func (data CreateMultisigData) Gas() int64 {
	return gasCreateMultisig
}

func (data CreateMultisigData) TxType() TxType {
	return TypeCreateMultisig
}

func (data CreateMultisigData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	lenWeights := len(data.Weights)
	if lenWeights > 32 {
		return &Response{
			Code: code.TooLargeOwnersList,
			Log:  "Owners list is limited to 32 items",
			Info: EncodeError(code.NewTooLargeOwnersList(strconv.Itoa(lenWeights), "32"))}
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

	return nil
}

func (data CreateMultisigData) String() string {
	return "CREATE MULTISIG"
}

func (data CreateMultisigData) CommissionData(price *commission.Price) *big.Int {
	return price.CreateMultisig
}

func (data CreateMultisigData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := tx.Commission(price)
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	msigAddress := accounts.CreateMultisigAddress(sender, tx.Nonce)

	if checkState.Accounts().ExistsMultisig(msigAddress) {
		return Response{
			Code: code.MultisigExists,
			Log:  fmt.Sprintf("Multisig %s already exists", msigAddress.String()),
			Info: EncodeError(code.NewMultisigExists(msigAddress.String())),
		}
	}
	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		var tagsCom *tagPoolChange
		if isGasCommissionFromPoolSwap {
			var (
				poolIDCom  uint32
				detailsCom *swap.ChangeDetailsWithOrders
				ownersCom  map[types.Address]*big.Int
			)
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.GasCoin,
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				Sellers:  make([]*OrderDetail, 0, len(ownersCom)),
			}
			for address, value := range ownersCom {
				deliverState.Accounts.AddBalance(address, tx.GasCoin, value)
				tagsCom.Sellers = append(tagsCom.Sellers, &OrderDetail{Owner: address, Value: value.String()})
			}
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		deliverState.Accounts.CreateMultisig(data.Weights, data.Addresses, data.Threshold, msigAddress)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.created_multisig"), Value: []byte(hex.EncodeToString(msigAddress[:])), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
