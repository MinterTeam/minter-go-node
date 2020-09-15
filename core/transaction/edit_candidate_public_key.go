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

type EditCandidatePublicKeyData struct {
	PubKey    types.Pubkey
	NewPubKey types.Pubkey
}

func (data EditCandidatePublicKeyData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCandidatePublicKeyData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data EditCandidatePublicKeyData) String() string {
	return fmt.Sprintf("EDIT CANDIDATE PUB KEY old: %x, new: %x",
		data.PubKey, data.NewPubKey)
}

func (data EditCandidatePublicKeyData) Gas() int64 {
	return commissions.EditCandidatePublicKey
}

func (data EditCandidatePublicKeyData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	if data.PubKey == data.NewPubKey {
		return Response{
			Code: code.NewPublicKeyIsBad,
			Log:  fmt.Sprintf("Current public key (%s) equals new public key (%s)", data.PubKey.String(), data.NewPubKey.String()),
			Info: EncodeError(code.NewNewPublicKeyIsBad(data.PubKey.String(), data.NewPubKey.String())),
		}
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

	if checkState.Candidates().IsBlockedPubKey(data.NewPubKey) {
		return Response{
			Code: code.PublicKeyInBlockList,
			Log:  fmt.Sprintf("Public key (%s) exists in block list", data.NewPubKey.String()),
			Info: EncodeError(code.NewCandidateExists(data.NewPubKey.String())),
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		deliverState.Coins.SubVolume(tx.GasCoin, commission)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliverState.Candidates.ChangePubKey(data.PubKey, data.NewPubKey)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditCandidatePublicKey)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
