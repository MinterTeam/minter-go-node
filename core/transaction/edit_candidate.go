package transaction

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type CandidateTx interface {
	GetPubKey() types.Pubkey
}

type EditCandidateData struct {
	PubKey        types.Pubkey
	RewardAddress types.Address
	OwnerAddress  types.Address
}

func (data EditCandidateData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey        string `json:"pub_key"`
		RewardAddress string `json:"reward_address"`
		OwnerAddress  string `json:"owner_address"`
	}{
		PubKey:        data.PubKey.String(),
		RewardAddress: data.RewardAddress.String(),
		OwnerAddress:  data.OwnerAddress.String(),
	})
}

func (data EditCandidateData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data EditCandidateData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data EditCandidateData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data EditCandidateData) String() string {
	return fmt.Sprintf("EDIT CANDIDATE pubkey: %x",
		data.PubKey)
}

func (data EditCandidateData) Gas() int64 {
	return commissions.EditCandidate
}

func (data EditCandidateData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !tx.GasCoin.IsBaseCoin() {
		coin := checkState.Coins().GetCoin(tx.GasCoin)

		errResp := CheckReserveUnderflow(coin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}

		if coin.Reserve().Cmp(commissionInBaseCoin) < 0 {
			return Response{
				Code: code.CoinReserveNotSufficient,
				Log:  fmt.Sprintf("Coin reserve balance is not sufficient for transaction. Has: %s, required %s", coin.Reserve().String(), commissionInBaseCoin.String()),
				Info: EncodeError(map[string]string{
					"has_reserve": coin.Reserve().String(),
					"commission":  commissionInBaseCoin.String(),
					"gas_coin":    coin.CName,
				}),
			}
		}

		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	if checkState.Accounts().GetBalance(sender, tx.GasCoin).Cmp(commission) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission.String(), tx.GasCoin),
			Info: EncodeError(map[string]string{
				"sender":       sender.String(),
				"needed_value": commission.String(),
				"gas_coin":     fmt.Sprintf("%s", tx.GasCoin),
			}),
		}
	}

	if deliveryState, ok := context.(*state.State); ok {
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliveryState.Coins.SubReserve(tx.GasCoin, commissionInBaseCoin)
		deliveryState.Coins.SubVolume(tx.GasCoin, commission)

		deliveryState.Accounts.SubBalance(sender, tx.GasCoin, commission)
		deliveryState.Candidates.Edit(data.PubKey, data.RewardAddress, data.OwnerAddress)
		deliveryState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeEditCandidate)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}

func checkCandidateOwnership(data CandidateTx, tx *Transaction, context *state.CheckState) *Response {
	if !context.Candidates().Exists(data.GetPubKey()) {
		return &Response{
			Code: code.CandidateNotFound,
			Log:  fmt.Sprintf("Candidate with such public key (%s) not found", data.GetPubKey().String()),
			Info: EncodeError(map[string]string{
				"public_key": data.GetPubKey().String(),
			}),
		}
	}

	owner := context.Candidates().GetCandidateOwner(data.GetPubKey())
	sender, _ := tx.Sender()
	if owner != sender {
		return &Response{
			Code: code.IsNotOwnerOfCandidate,
			Log:  fmt.Sprintf("Sender is not an owner of a candidate")}
	}

	return nil
}
