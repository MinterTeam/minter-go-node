package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type VoteUpdateDataV230 struct {
	Version string
	PubKey  types.Pubkey
	Height  uint64
}

func (data VoteUpdateDataV230) Gas() int64 {
	return gasVoteUpdate
}
func (data VoteUpdateDataV230) TxType() TxType {
	return TypeVoteUpdate
}

func (data VoteUpdateDataV230) GetPubKey() types.Pubkey {
	return data.PubKey
}

var allowedVersionNameRegexpCompile, _ = regexp.Compile("^[a-zA-Z0-9_]{1,20}$")

func (data VoteUpdateDataV230) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	if !allowedVersionNameRegexpCompile.Match([]byte(data.Version)) {
		return &Response{
			Code: code.WrongUpdateVersionName,
			Log:  "wrong version name",
			Info: EncodeError(code.NewCustomCode(code.WrongUpdateVersionName)),
		}
	}

	if data.Height < block {
		return &Response{
			Code: code.VoteExpired,
			Log:  "vote is produced for the past state",
			Info: EncodeError(code.NewVoteExpired(strconv.Itoa(int(block)), strconv.Itoa(int(data.Height)))),
		}
	}

	if context.Updates().IsVoteExists(data.Height, data.PubKey) {
		return &Response{
			Code: code.VoteAlreadyExists,
			Log:  "Update vote with such public key and height already exists",
			Info: EncodeError(code.NewVoteAlreadyExists(strconv.FormatUint(data.Height, 10), data.GetPubKey().String())),
		}
	}
	return checkCandidateOwnership(data, tx, context)
}

func (data VoteUpdateDataV230) String() string {
	return fmt.Sprintf("UPDATE NETWORK on height: %d", data.Height)
}

func (data VoteUpdateDataV230) CommissionData(price *commission.Price) *big.Int {
	return price.VoteUpdate
}

func (data VoteUpdateDataV230) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
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

		deliverState.Updates.AddVote(data.Height, data.PubKey, data.Version)

		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.public_key"), Value: []byte(hex.EncodeToString(data.PubKey[:])), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
