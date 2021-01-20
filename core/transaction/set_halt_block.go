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
	"github.com/MinterTeam/minter-go-node/hexutil"
	"github.com/tendermint/tendermint/libs/kv"
)

type SetHaltBlockData struct {
	PubKey types.Pubkey
	Height uint64
}

func (data SetHaltBlockData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data SetHaltBlockData) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	if data.Height < block {
		return &Response{
			Code: code.VoiceExpired,
			Log:  fmt.Sprintf("Halt height should be equal or bigger than current: %d", block),
			Info: EncodeError(code.NewVoiceExpired(strconv.FormatUint(data.Height, 10), data.GetPubKey().String())),
		}
	}

	if context.Halts().IsHaltExists(data.Height, data.PubKey) {
		return &Response{
			Code: code.HaltAlreadyExists,
			Log:  "Halt with such public key and height already exists",
			Info: EncodeError(code.NewHaltAlreadyExists(strconv.FormatUint(data.Height, 10), data.GetPubKey().String())),
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

func (data SetHaltBlockData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, priceCoin types.CoinID, price *big.Int) Response {
	sender, _ := tx.Sender()

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	response := data.basicCheck(tx, checkState, currentBlock)
	if response != nil {
		return *response
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		gasCoin := checkState.Coins().GetCoin(tx.GasCoin)

		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, gasCoin.GetFullSymbol()),
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
		deliverState.Halts.AddHaltBlock(data.Height, data.PubKey)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
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
