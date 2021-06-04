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
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/hexutil"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type SetHaltBlockData struct {
	PubKey types.Pubkey
	Height uint64
}

func (data SetHaltBlockData) Gas() int64 {
	return gasSetHaltBlock
}

func (data SetHaltBlockData) TxType() TxType {
	return TypeSetHaltBlock
}

func (data SetHaltBlockData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data SetHaltBlockData) basicCheck(tx *Transaction, context *state.CheckState, block uint64) *Response {
	if data.Height < block {
		return &Response{
			Code: code.VoteExpired,
			Log:  fmt.Sprintf("Halt height should be equal or bigger than current: %d", block),
			Info: EncodeError(code.NewVoteExpired(strconv.FormatUint(data.Height, 10), data.GetPubKey().String())),
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

func (data SetHaltBlockData) CommissionData(price *commission.Price) *big.Int {
	return price.SetHaltBlock
}

func (data SetHaltBlockData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := tx.Commission(price)
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
		deliverState.Halts.AddHaltBlock(data.Height, data.PubKey)
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
