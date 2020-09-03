package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
	"strconv"
)

type SetHaltBlockData struct {
	PubKey types.Pubkey
	Height uint64
}

func (data SetHaltBlockData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data SetHaltBlockData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if !context.Candidates().Exists(data.PubKey) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key not found"),
			Info: EncodeError(map[string]string{
				"pub_key": data.PubKey.String(),
			}),
		}
	}

	return checkCandidateOwnership(data, tx, context)
}

func (data SetHaltBlockData) String() string {
	return fmt.Sprintf("SET HALT BLOCK pubkey:%s height:%d",
		hexutil.Encode(data.PubKey[:]), data.Height)
}

func (data SetHaltBlockData) Gas() int64 {
	return commissions.SetHaltBlock
}

func (data SetHaltBlockData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	if data.Height < currentBlock {
		return Response{
			Code: code.WrongHaltHeight,
			Log:  fmt.Sprintf("Halt height should be equal or bigger than current: %d", currentBlock),
			Info: EncodeError(map[string]string{
				"height": strconv.FormatUint(data.Height, 10),
			}),
		}
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

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
					"has_value":      gasCoin.Reserve().String(),
					"required_value": commissionInBaseCoin.String(),
					"coin":           gasCoin.GetFullSymbol(),
				}),
			}
		}

		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

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

	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		deliverState.Coins.SubVolume(tx.GasCoin, commission)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Halts.AddHaltBlock(data.Height, data.PubKey)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSetHaltBlock)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
