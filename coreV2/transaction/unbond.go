package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/hexutil"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	"math/big"
)

type UnbondData struct {
	PubKey types.Pubkey
	Coin   types.CoinID
	Value  *big.Int
}

func (data UnbondData) Gas() int64 {
	return gasUnbond
}
func (data UnbondData) TxType() TxType {
	return TypeUnbond
}

func (data UnbondData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Value == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if !context.Coins().Exists(data.Coin) {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Coin),
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !context.Candidates().Exists(data.PubKey) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  "Candidate with such public key not found",
			Info: EncodeError(code.NewCandidateNotFound(data.PubKey.String())),
		}
	}

	sender, _ := tx.Sender()

	if waitlist := context.WaitList().Get(sender, data.PubKey, data.Coin); waitlist != nil {
		if data.Value.Cmp(waitlist.Value) != 1 {
			return nil
		}
		return &Response{
			Code: code.InsufficientWaitList,
			Log:  "Insufficient amount at waitlist for sender account",
			Info: EncodeError(code.NewInsufficientWaitList(waitlist.Value.String(), data.Value.String())),
		}
	}

	stake := context.Candidates().GetStakeValueOfAddress(data.PubKey, sender, data.Coin)

	if stake == nil {
		return &Response{
			Code: code.StakeNotFound,
			Log:  "Stake of current user not found",
			Info: EncodeError(code.NewStakeNotFound(data.PubKey.String(), sender.String(), data.Coin.String(), context.Coins().GetCoin(data.Coin).GetFullSymbol())),
		}
	}

	if stake.Cmp(data.Value) < 0 {
		return &Response{
			Code: code.InsufficientStake,
			Log:  "Insufficient stake for sender account",
			Info: EncodeError(code.NewInsufficientStake(data.PubKey.String(), sender.String(), data.Coin.String(), context.Coins().GetCoin(data.Coin).GetFullSymbol(), stake.String(), data.Value.String())),
		}
	}

	return nil
}

func (data UnbondData) String() string {
	return fmt.Sprintf("UNBOND pubkey:%s",
		hexutil.Encode(data.PubKey[:]))
}

func (data UnbondData) CommissionData(price *commission.Price) *big.Int {
	return price.Unbond
}

func (data UnbondData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	var tags []abcTypes.EventAttribute
	if deliverState, ok := context.(*state.State); ok {
		// now + 30 days
		unbondAtBlock := currentBlock + types.GetUnbondPeriod()

		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin, _ = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else if !tx.GasCoin.IsBaseCoin() {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		if waitList := deliverState.Waitlist.Get(sender, data.PubKey, data.Coin); waitList != nil {
			diffValue := big.NewInt(0).Sub(data.Value, waitList.Value)
			deliverState.Waitlist.Delete(sender, data.PubKey, data.Coin)
			if diffValue.Sign() == -1 {
				deliverState.Waitlist.AddWaitList(sender, data.PubKey, data.Coin, big.NewInt(0).Neg(diffValue))
			}
		} else {
			deliverState.Candidates.SubStake(sender, data.PubKey, data.Coin, data.Value)
		}

		deliverState.FrozenFunds.AddFund(unbondAtBlock, sender, data.PubKey, deliverState.Candidates.ID(data.PubKey), data.Coin, data.Value, nil)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.public_key"), Value: []byte(hex.EncodeToString(data.PubKey[:])), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
