package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type EditCandidateCommission struct {
	PubKey     types.Pubkey
	Commission uint32
}

func (data EditCandidateCommission) Gas() int64 {
	return gasEditCandidateCommission
}
func (data EditCandidateCommission) TxType() TxType {
	return TypeEditCandidateCommission
}

func (data EditCandidateCommission) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCandidateCommission) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	errResp := checkCandidateOwnership(data, tx, context)
	if errResp != nil {
		return errResp
	}

	candidate := context.Candidates().GetCandidate(data.PubKey)

	maxNewCommission, minNewCommission := candidate.Commission+10, candidate.Commission-10
	if maxNewCommission > maxCommission {
		maxNewCommission = maxCommission
	}
	if minNewCommission < minCommission || minNewCommission > maxCommission {
		minNewCommission = minCommission
	}
	if data.Commission < minNewCommission || data.Commission > maxNewCommission {
		return &Response{
			Code: code.WrongCommission,
			Log:  fmt.Sprintf("You want change commission from %d to %d, but you can change no more than 10 units, because commission should be between %d and %d", candidate.Commission, data.Commission, minNewCommission, maxNewCommission),
			Info: EncodeError(code.NewWrongCommission(fmt.Sprintf("%d", data.Commission), strconv.Itoa(int(minNewCommission)), strconv.Itoa(int(maxNewCommission)))),
		}
	}

	if candidate.LastEditCommissionHeight+3*types.GetUnbondPeriod() > block {
		return &Response{
			Code: code.PeriodLimitReached,
			Log:  fmt.Sprintf("You cannot change the commission more than once every %d blocks, the last change was on block %d", 3*types.GetUnbondPeriod(), candidate.LastEditCommissionHeight),
			Info: EncodeError(code.NewPeriodLimitReached(strconv.Itoa(int(candidate.LastEditCommissionHeight+3*types.GetUnbondPeriod())), strconv.Itoa(int(candidate.LastEditCommissionHeight)))),
		}
	}

	return nil
}

func (data EditCandidateCommission) String() string {
	return fmt.Sprintf("EDIT COMMISSION: %s", data.PubKey)
}

func (data EditCandidateCommission) CommissionData(price *commission.Price) *big.Int {
	return price.EditCandidateCommission
}

func (data EditCandidateCommission) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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
			commission, commissionInBaseCoin, poolIDCom, detailsCom, ownersCom = deliverState.Swap.PairSellWithOrders(tx.CommissionCoin(), types.GetBaseCoinID(), commission, big.NewInt(0))
			tagsCom = &tagPoolChange{
				PoolID:   poolIDCom,
				CoinIn:   tx.CommissionCoin(),
				ValueIn:  commission.String(),
				CoinOut:  types.GetBaseCoinID(),
				ValueOut: commissionInBaseCoin.String(),
				Orders:   detailsCom,
				// Sellers:  ownersCom,
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

		deliverState.Candidates.EditCommission(data.PubKey, data.Commission, currentBlock)
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
