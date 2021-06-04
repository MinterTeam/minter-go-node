package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/state/swap"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	abcTypes "github.com/tendermint/tendermint/abci/types"
)

type EditCandidatePublicKeyData struct {
	PubKey    types.Pubkey
	NewPubKey types.Pubkey
}

func (data EditCandidatePublicKeyData) Gas() int64 {
	return gasEditCandidatePublicKey
}
func (data EditCandidatePublicKeyData) TxType() TxType {
	return TypeEditCandidatePublicKey
}

func (data EditCandidatePublicKeyData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCandidatePublicKeyData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data EditCandidatePublicKeyData) String() string {
	return fmt.Sprintf("EDIT CANDIDATE PUB KEY old: %x, new: %x",
		data.PubKey, data.NewPubKey)
}

func (data EditCandidatePublicKeyData) CommissionData(price *commission.Price) *big.Int {
	return price.EditCandidatePublicKey
}

func (data EditCandidatePublicKeyData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	if data.PubKey == data.NewPubKey {
		return Response{
			Code: code.NewPublicKeyIsBad,
			Log:  fmt.Sprintf("Current public key (%s) equals new public key (%s)", data.PubKey.String(), data.NewPubKey.String()),
			Info: EncodeError(code.NewNewPublicKeyIsBad(data.PubKey.String(), data.NewPubKey.String())),
		}
	}

	if checkState.Candidates().Exists(data.NewPubKey) {
		return Response{
			Code: code.CandidateExists,
			Log:  fmt.Sprintf("Candidate with such public key (%s) already exists", data.NewPubKey.String()),
			Info: EncodeError(code.NewCandidateExists(data.NewPubKey.String())),
		}
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
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), gasCoin.GetFullSymbol()),
			Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
		}
	}

	if checkState.Candidates().IsBlockedPubKey(data.NewPubKey) {
		return Response{
			Code: code.PublicKeyInBlockList,
			Log:  fmt.Sprintf("Public key (%s) exists in block list", data.NewPubKey.String()),
			Info: EncodeError(code.NewPublicKeyInBlockList(data.NewPubKey.String())),
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

		deliverState.Candidates.ChangePubKey(data.PubKey, data.NewPubKey)

		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.public_key"), Value: []byte(hex.EncodeToString(data.PubKey[:])), Index: true},
			{Key: []byte("tx.new_public_key"), Value: []byte(hex.EncodeToString(data.NewPubKey[:])), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
