package transaction

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/crypto/sha3"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/libs/kv"
	"math/big"
)

type RedeemCheckData struct {
	RawCheck []byte
	Proof    [65]byte
}

func (data RedeemCheckData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RawCheck string `json:"raw_check"`
		Proof    string `json:"proof"`
	}{
		RawCheck: base64.StdEncoding.EncodeToString(data.RawCheck),
		Proof:    base64.StdEncoding.EncodeToString(data.Proof[:]),
	})
}

func (data RedeemCheckData) TotalSpend(tx *Transaction, context *state.CheckState) (TotalSpends, []Conversion, *big.Int, *Response) {
	panic("implement me")
}

func (data RedeemCheckData) CommissionInBaseCoin(tx *Transaction) *big.Int {
	panic("implement me")
}

func (data RedeemCheckData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.RawCheck == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data"}
	}

	// fixed potential problem with making too high commission for sender
	if tx.GasPrice != 1 {
		return &Response{
			Code: code.TooHighGasPrice,
			Log:  fmt.Sprintf("Gas price for check is limited to 1")}
	}

	return nil
}

func (data RedeemCheckData) String() string {
	return fmt.Sprintf("REDEEM CHECK proof: %x", data.Proof)
}

func (data RedeemCheckData) Gas() int64 {
	return commissions.RedeemCheckTx
}

func (data RedeemCheckData) Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64) Response {
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

	decodedCheck, err := check.DecodeFromBytes(data.RawCheck)
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
		}
	}

	if decodedCheck.ChainID != types.CurrentChainID {
		return Response{
			Code: code.WrongChainID,
			Log:  "Wrong chain id",
			Info: EncodeError(map[string]string{
				"current_chain_id": fmt.Sprintf("%d", types.CurrentChainID),
				"got_chain_id":     fmt.Sprintf("%d", decodedCheck.ChainID),
			}),
		}
	}

	if len(decodedCheck.Nonce) > 16 {
		return Response{
			Code: code.TooLongNonce,
			Log:  "Nonce is too big. Should be up to 16 bytes.",
		}
	}

	checkSender, err := decodedCheck.Sender()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	if !checkState.Coins().Exists(decodedCheck.Coin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists"),
			Info: EncodeError(map[string]string{
				"coin": fmt.Sprintf("%s", decodedCheck.Coin),
			}),
		}
	}

	if !checkState.Coins().Exists(decodedCheck.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Gas coin not exists"),
			Info: EncodeError(map[string]string{
				"gas_coin": fmt.Sprintf("%s", decodedCheck.GasCoin),
			}),
		}
	}

	if tx.GasCoin != decodedCheck.GasCoin {
		return Response{
			Code: code.WrongGasCoin,
			Log:  fmt.Sprintf("Gas coin for redeem check transaction can only be %s", decodedCheck.GasCoin),
			Info: EncodeError(map[string]string{
				"gas_coin": fmt.Sprintf("%s", decodedCheck.GasCoin),
			}),
		}
	}

	if decodedCheck.DueBlock < currentBlock {
		return Response{
			Code: code.CheckExpired,
			Log:  fmt.Sprintf("Check expired"),
			Info: EncodeError(map[string]string{
				"due_block":     fmt.Sprintf("%d", decodedCheck.DueBlock),
				"current_block": fmt.Sprintf("%d", currentBlock),
			}),
		}
	}

	if checkState.Checks().IsCheckUsed(decodedCheck) {
		return Response{
			Code: code.CheckUsed,
			Log:  fmt.Sprintf("Check already redeemed")}
	}

	lockPublicKey, err := decodedCheck.LockPubKey()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
		}
	}

	var senderAddressHash types.Hash
	hw := sha3.NewKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		sender,
	})
	hw.Sum(senderAddressHash[:0])

	pub, err := crypto.Ecrecover(senderAddressHash[:], data.Proof[:])

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
		}
	}

	if !bytes.Equal(lockPublicKey, pub) {
		return Response{
			Code: code.CheckInvalidLock,
			Log:  "Invalid proof",
		}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(big.NewInt(int64(tx.GasPrice)), big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if !decodedCheck.GasCoin.IsBaseCoin() {
		coin := checkState.Coins().GetCoin(decodedCheck.GasCoin)
		errResp := CheckReserveUnderflow(coin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}
		commission = formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), commissionInBaseCoin)
	}

	if decodedCheck.Coin == decodedCheck.GasCoin {
		totalTxCost := big.NewInt(0).Add(decodedCheck.Value, commission)
		if checkState.Accounts().GetBalance(checkSender, decodedCheck.Coin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s %s. Wanted %s %s", decodedCheck.Coin, checkSender.String(), totalTxCost.String(), decodedCheck.Coin),
				Info: EncodeError(map[string]string{
					"sender":        checkSender.String(),
					"coin":          fmt.Sprintf("%s", decodedCheck.Coin),
					"total_tx_cost": totalTxCost.String(),
				}),
			}
		}
	} else {
		if checkState.Accounts().GetBalance(checkSender, decodedCheck.Coin).Cmp(decodedCheck.Value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s %s. Wanted %s %s", checkSender.String(), decodedCheck.Coin, decodedCheck.Value.String(), decodedCheck.Coin),
				Info: EncodeError(map[string]string{
					"sender": checkSender.String(),
					"coin":   fmt.Sprintf("%s", decodedCheck.Coin),
					"value":  decodedCheck.Value.String(),
				}),
			}
		}

		if checkState.Accounts().GetBalance(checkSender, decodedCheck.GasCoin).Cmp(commission) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s %s. Wanted %s %s", checkSender.String(), decodedCheck.GasCoin, commission.String(), decodedCheck.GasCoin),
				Info: EncodeError(map[string]string{
					"sender":     sender.String(),
					"gas_coin":   fmt.Sprintf("%s", decodedCheck.GasCoin),
					"commission": commission.String(),
				}),
			}
		}
	}

	if deliveryState, ok := context.(*state.State); ok {
		deliveryState.Checks.UseCheck(decodedCheck)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliveryState.Coins.SubVolume(decodedCheck.GasCoin, commission)
		deliveryState.Coins.SubReserve(decodedCheck.GasCoin, commissionInBaseCoin)

		deliveryState.Accounts.SubBalance(checkSender, decodedCheck.GasCoin, commission)
		deliveryState.Accounts.SubBalance(checkSender, decodedCheck.Coin, decodedCheck.Value)
		deliveryState.Accounts.AddBalance(sender, decodedCheck.Coin, decodedCheck.Value)
		deliveryState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeRedeemCheck)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(checkSender[:]))},
		kv.Pair{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin"), Value: []byte(decodedCheck.Coin.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
