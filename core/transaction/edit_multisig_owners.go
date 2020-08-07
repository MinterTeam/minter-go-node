package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type EditMultisigOwnersData struct {
	MultisigAddress types.Address
	Weights         []uint
	Addresses       []types.Address
}

func (data EditMultisigOwnersData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if !context.Accounts().GetAccount(data.MultisigAddress).IsMultisig() {
		return &Response{
			Code: code.MultisigNotExists,
			Log:  "Multisig does not exists",
			Info: EncodeError(map[string]string{
				"multisig_address": data.MultisigAddress.String(),
			}),
		}
	}

	return nil
}

func (data EditMultisigOwnersData) String() string {
	return fmt.Sprintf("EDIT MULTISIG OWNERS address: %x", data.MultisigAddress)
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
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(map[string]string{
				"sender":       sender.String(),
				"needed_value": commission.String(),
				"gas_coin":     gasCoin.GetFullSymbol(),
			}),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(tx.GasCoin, commission)
		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		deliverState.Accounts.EditMultisig(data.Weights, data.Addresses, sender)
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
