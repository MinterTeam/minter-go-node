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
	"github.com/MinterTeam/minter-go-node/coreV2/validators"
	abcTypes "github.com/tendermint/tendermint/abci/types"
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

func (data DeclareCandidacyData) Gas() int64 {
	return gasDeclareCandidacy
}
func (data DeclareCandidacyData) TxType() TxType {
	return TypeDeclareCandidacy
}

func (data DeclareCandidacyData) basicCheck(tx *Transaction, context *state.CheckState) *Response {
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
			Log:  "coin has no reserve",
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

func (data DeclareCandidacyData) CommissionData(price *commission.Price) *big.Int {
	return price.DeclareCandidacy
}

func (data DeclareCandidacyData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response {
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

	commissionInBaseCoin := price
	commissionPoolSwapper := checkState.Swap().GetSwapper(tx.GasCoin, types.GetBaseCoinID())
	gasCoin := checkState.Coins().GetCoin(tx.GasCoin)
	commission, isGasCommissionFromPoolSwap, errResp := CalculateCommission(checkState, commissionPoolSwapper, gasCoin, commissionInBaseCoin)
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

		deliverState.Accounts.SubBalance(sender, data.Coin, data.Stake)
		deliverState.Candidates.Create(data.Address, sender, sender, data.PubKey, data.Commission, currentBlock, 0)
		deliverState.Candidates.Delegate(sender, data.PubKey, data.Coin, data.Stake, big.NewInt(0))
		deliverState.Accounts.SetNonce(sender, tx.Nonce)

		tags = []abcTypes.EventAttribute{
			{Key: []byte("tx.commission_in_base_coin"), Value: []byte(commissionInBaseCoin.String())},
			{Key: []byte("tx.commission_conversion"), Value: []byte(isGasCommissionFromPoolSwap.String()), Index: true},
			{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
			{Key: []byte("tx.commission_details"), Value: []byte(tagsCom.string())},
			{Key: []byte("tx.public_key"), Value: []byte(hex.EncodeToString(data.PubKey[:])), Index: true},
			{Key: []byte("tx.coin_id"), Value: []byte(data.Coin.String()), Index: true},
		}
	}

	return Response{
		Code: code.OK,
		Tags: tags,
	}
}
