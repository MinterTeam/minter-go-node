package transaction

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/tendermint/libs/kv"
	"golang.org/x/crypto/sha3"
)

type RedeemCheckData struct {
	RawCheck []byte
	Proof    [65]byte
}

func (data RedeemCheckData) BasicCheck(tx *Transaction, context *state.CheckState) *Response {
	if data.RawCheck == nil {
		return &Response{
			Code: code.DecodeError,
			Log:  "Incorrect tx data",
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	// fixed potential problem with making too high commission for sender
	if tx.GasPrice != 1 {
		return &Response{
			Code: code.TooHighGasPrice,
			Log:  "Gas price for check is limited to 1",
			Info: EncodeError(code.NewTooHighGasPrice("1", strconv.Itoa(int(tx.GasPrice)))),
		}
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
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if decodedCheck.ChainID != types.CurrentChainID {
		return Response{
			Code: code.WrongChainID,
			Log:  "Wrong chain id",
			Info: EncodeError(code.NewWrongChainID(fmt.Sprintf("%d", types.CurrentChainID), fmt.Sprintf("%d", tx.ChainID))),
		}
	}

	if len(decodedCheck.Nonce) > 16 {
		return Response{
			Code: code.TooLongNonce,
			Log:  "Nonce is too big. Should be up to 16 bytes.",
			Info: EncodeError(code.NewTooLongNonce(strconv.Itoa(len(decodedCheck.Nonce)), "16")),
		}
	}

	checkSender, err := decodedCheck.Sender()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if !checkState.Coins().Exists(decodedCheck.Coin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  "Coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", decodedCheck.Coin.String())),
		}
	}

	if !checkState.Coins().Exists(decodedCheck.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  "Gas coin not exists",
			Info: EncodeError(code.NewCoinNotExists("", decodedCheck.GasCoin.String())),
		}
	}

	if tx.GasCoin != decodedCheck.GasCoin {
		return Response{
			Code: code.WrongGasCoin,
			Log:  fmt.Sprintf("Gas coin for redeem check transaction can only be %s", decodedCheck.GasCoin),
			Info: EncodeError(code.NewWrongGasCoin(checkState.Coins().GetCoin(tx.GasCoin).GetFullSymbol(), tx.GasCoin.String(), checkState.Coins().GetCoin(decodedCheck.GasCoin).GetFullSymbol(), decodedCheck.GasCoin.String())),
		}
	}

	if decodedCheck.DueBlock < currentBlock {
		return Response{
			Code: code.CheckExpired,
			Log:  "Check expired",
			Info: EncodeError(code.MewCheckExpired(fmt.Sprintf("%d", decodedCheck.DueBlock), fmt.Sprintf("%d", currentBlock))),
		}
	}

	if checkState.Checks().IsCheckUsed(decodedCheck) {
		return Response{
			Code: code.CheckUsed,
			Log:  "Check already redeemed",
			Info: EncodeError(code.NewCheckUsed()),
		}
	}

	lockPublicKey, err := decodedCheck.LockPubKey()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	var senderAddressHash types.Hash
	hw := sha3.NewLegacyKeccak256()
	_ = rlp.Encode(hw, []interface{}{
		sender,
	})
	hw.Sum(senderAddressHash[:0])

	pub, err := crypto.Ecrecover(senderAddressHash[:], data.Proof[:])

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if !bytes.Equal(lockPublicKey, pub) {
		return Response{
			Code: code.CheckInvalidLock,
			Log:  "Invalid proof",
			Info: EncodeError(code.NewCheckInvalidLock()),
		}
	}

	commissionInBaseCoin := tx.CommissionInBaseCoin()
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	gasCoin := checkState.Coins().GetCoin(decodedCheck.GasCoin)
	coin := checkState.Coins().GetCoin(decodedCheck.Coin)

	if !decodedCheck.GasCoin.IsBaseCoin() {
		errResp := CheckReserveUnderflow(gasCoin, commissionInBaseCoin)
		if errResp != nil {
			return *errResp
		}
		commission = formula.CalculateSaleAmount(gasCoin.Volume(), gasCoin.Reserve(), gasCoin.Crr(), commissionInBaseCoin)
	}

	if decodedCheck.Coin == decodedCheck.GasCoin {
		totalTxCost := big.NewInt(0).Add(decodedCheck.Value, commission)
		if checkState.Accounts().GetBalance(checkSender, decodedCheck.Coin).Cmp(totalTxCost) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s %s. Wanted %s %s", decodedCheck.Coin, checkSender.String(), totalTxCost.String(), coin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), totalTxCost.String(), coin.GetFullSymbol(), coin.ID().String())),
			}
		}
	} else {
		if checkState.Accounts().GetBalance(checkSender, decodedCheck.Coin).Cmp(decodedCheck.Value) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s %s. Wanted %s %s", checkSender.String(), decodedCheck.Coin, decodedCheck.Value.String(), coin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(checkSender.String(), decodedCheck.Value.String(), coin.GetFullSymbol(), coin.ID().String())),
			}
		}

		if checkState.Accounts().GetBalance(checkSender, decodedCheck.GasCoin).Cmp(commission) < 0 {
			return Response{
				Code: code.InsufficientFunds,
				Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s %s. Wanted %s %s", checkSender.String(), decodedCheck.GasCoin, commission.String(), gasCoin.GetFullSymbol()),
				Info: EncodeError(code.NewInsufficientFunds(sender.String(), commission.String(), gasCoin.GetFullSymbol(), gasCoin.ID().String())),
			}
		}
	}

	if deliverState, ok := context.(*state.State); ok {
		deliverState.Checks.UseCheck(decodedCheck)
		rewardPool.Add(rewardPool, commissionInBaseCoin)

		deliverState.Coins.SubVolume(decodedCheck.GasCoin, commission)
		deliverState.Coins.SubReserve(decodedCheck.GasCoin, commissionInBaseCoin)

		deliverState.Accounts.SubBalance(checkSender, decodedCheck.GasCoin, commission)
		deliverState.Accounts.SubBalance(checkSender, decodedCheck.Coin, decodedCheck.Value)
		deliverState.Accounts.AddBalance(sender, decodedCheck.Coin, decodedCheck.Value)
		deliverState.Accounts.SetNonce(sender, tx.Nonce)
	}

	tags := kv.Pairs{
		kv.Pair{Key: []byte("tx.type"), Value: []byte(hex.EncodeToString([]byte{byte(TypeRedeemCheck)}))},
		kv.Pair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(checkSender[:]))},
		kv.Pair{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(sender[:]))},
		kv.Pair{Key: []byte("tx.coin_id"), Value: []byte(decodedCheck.Coin.String())},
		kv.Pair{Key: []byte("tx.commission_amount"), Value: []byte(commission.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
