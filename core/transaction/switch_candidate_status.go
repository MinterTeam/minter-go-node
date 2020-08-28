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

type SetCandidateOnData struct {
	PubKey types.Pubkey
}

func (data SetCandidateOnData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pub_key"`
	}{
		PubKey: data.PubKey.String(),
	})
}

func (data SetCandidateOnData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data SetCandidateOnData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data SetCandidateOnData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data SetCandidateOnData) String() string {
	return fmt.Sprintf("SET CANDIDATE ONLINE pubkey: %x",
		data.PubKey)
}

func (data SetCandidateOnData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

func (data SetCandidateOnData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin),
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
		deliveryState.Candidates.SetOnline(data.PubKey)
		deliveryState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSetCandidateOnline)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}

type SetCandidateOffData struct {
	PubKey types.Pubkey `json:"pub_key"`
}

func (data SetCandidateOffData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string `json:"pub_key"`
	}{
		PubKey: data.PubKey.String(),
	})
}

func (data SetCandidateOffData) GetPubKey() types.Pubkey {
	return data.PubKey
}

func (data SetCandidateOffData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data SetCandidateOffData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	return checkCandidateOwnership(data, tx, context)
}

func (data SetCandidateOffData) String() string {
	return fmt.Sprintf("SET CANDIDATE OFFLINE pubkey: %x",
		data.PubKey)
}

func (data SetCandidateOffData) Gas() int64 {
	return commissions.ToggleCandidateStatus
}

func (data SetCandidateOffData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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
			Log:  fmt.Sprintf("Insufficient funds for sender account: %s. Wanted %s %s", sender.String(), commission, tx.GasCoin),
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
		deliveryState.Candidates.SetOffline(data.PubKey)
		deliveryState.Validators.SetToDrop(data.PubKey)
		deliveryState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeSetCandidateOffline)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(sender[:]))},
	}

	return Response{
		Code:      code.OK,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
		Tags:      tags,
	}
}
