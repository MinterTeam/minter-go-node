package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

const minCommission = 0
const maxCommission = 100

type DeclareCandidacyData struct {
	Address    types.Address
	PubKey     types.Pubkey
	Commission uint32
	Coin       types.CoinID
	Stake      *big.Int
}

func (data DeclareCandidacyData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.Stake == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	coin := context.Coins().GetCoin(data.Coin)
	if coin == nil {
		return &Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", data.Coin),
			Info: EncodeError(code.NewCoinNotExists("", data.Coin.String())),
		}
	}

	if !coin.BaseOrHasReserve() {
		return &Response{
			Code: code.CoinHasNotReserve,
			Log:  "coin has not reserve",
			Info: EncodeError(code.NewCoinHasNotReserve(
				coin.GetFullSymbol(),
				coin.ID().String(),
			)),
		}
	}

	if context.Candidates().Exists(data.PubKey) {
		return &Response{
			Code: code.CandidateExists,
			Log:  fmt.Sprintf("Candidate with such public key (%s) already exists", data.PubKey.String()),
			Info: EncodeError(code.NewCandidateExists(data.PubKey.String())),
		}
	}

	if context.Candidates().IsBlockedPubKey(data.PubKey) {
		return &Response{
			Code: code.PublicKeyInBlockList,
			Log:  fmt.Sprintf("Candidate with such public key (%s) exists in block list", data.PubKey.String()),
			Info: EncodeError(code.NewPublicKeyInBlockList(data.PubKey.String())),
		}
	}

	if data.Commission < minCommission || data.Commission > maxCommission {
		return &Response{
			Code: code.WrongCommission,
			Log:  "Commission should be between 0 and 100",
			Info: EncodeError(code.NewWrongCommission(fmt.Sprintf("%d", data.Commission), "0", "100")),
		}
	}

	return nil
}

func (data DeclareCandidacyData) String() string {
	return fmt.Sprintf("DECLARE CANDIDACY address:%s pubkey:%s commission: %d",
		data.Address.String(), data.PubKey.String(), data.Commission)
}

func (data DeclareCandidacyData) Gas() int64 {
	return commissions.DeclareCandidacyTx
}

func (data DeclareCandidacyData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	maxCandidatesCount := validators.GetCandidatesCountForBlock(currentBlock)

	if checkState.Candidates().Count() >= maxCandidatesCount && !checkState.Candidates().IsNewCandidateStakeSufficient(data.Coin, data.Stake, maxCandidatesCount) {
		return Response{
			Code: code.TooLowStake,
			Log:  "Given stake is too low",
			Info: EncodeError(code.NewTooLowStake(sender.String(), data.PubKey.String(), data.Stake.String(), data.Coin.String(), checkState.Coins().GetCoin(data.Coin).GetFullSymbol())),
		}
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, gasCoin, commissionInBaseCoin)
	if errResp != nil {
		return *errResp
	}

	if checkState.Accounts().GetBalance(sender, data.Coin).Cmp(data.Stake) < 0 {
		coin := checkState.Coins().GetCoin(data.Coin)
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), data.Stake, coin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), data.Stake.String(), coin.GetFullSymbol(), coin.ID().String())),
		}
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if data.Coin == tx.GasCoin {
		totalTxCost := big.NewInt(0)
		totalTxCost.Add(totalTxCost, data.Stake)
		totalTxCost.Add(totalTxCost, commission)

		if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), totalTxCost.String(), gasCoin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), totalTxCost.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
			}
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		if isGasCommissionFromPoolSwap {
			commission, commissionInBaseCoin = deliverState.Swap.PairSell(tx.GasCoin, types.GetBaseCoinID(), commission, commissionInBaseCoin)
		} else {
			deliverState.Coins.SubVolume(tx.GasCoin, commission)
			deliverState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		}
		deliverState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(sender, data.Coin, data.Stake)
		deliverState.Candidates.Create(data.Address, sender, sender, data.PubKey, data.Commission, currentBlock)
		deliverState.Candidates.Delegate(sender, data.PubKey, data.Coin, data.Stake, big.NewInt(0))
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeDeclareCandidacy)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
