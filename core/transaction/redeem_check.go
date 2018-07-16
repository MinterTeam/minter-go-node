package transaction

import (
	"bytes"
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
	"github.com/tendermint/tendermint/libs/common"
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
		RawCheck: fmt.Sprintf("Mc%x", data.RawCheck),
		Proof:    fmt.Sprintf("%x", data.Proof),
	})
}

func (data RedeemCheckData) String() string {
	return fmt.Sprintf("REDEEM CHECK proof: %x", data.Proof)
}

func (data RedeemCheckData) Gas() int64 {
	return commissions.SendTx
}

func (data RedeemCheckData) Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPull *big.Int, currentBlock uint64) Response {
	decodedCheck, err := check.DecodeFromBytes(data.RawCheck)

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	checkSender, err := decodedCheck.Sender()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	if !context.CoinExists(decodedCheck.Coin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin not exists")}
	}

	if decodedCheck.DueBlock < uint64(currentBlock) {
		return Response{
			Code: code.CheckExpired,
			Log:  fmt.Sprintf("Check expired")}
	}

	if context.IsCheckUsed(decodedCheck) {
		return Response{
			Code: code.CheckUsed,
			Log:  fmt.Sprintf("Check already redeemed")}
	}

	// fixed potential problem with making too high commission for sender
	if tx.GasPrice.Cmp(big.NewInt(1)) == 1 {
		return Response{
			Code: code.TooHighGasPrice,
			Log:  fmt.Sprintf("Gas price for check is limited to 1")}
	}

	lockPublicKey, err := decodedCheck.LockPubKey()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	var senderAddressHash types.Hash
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, []interface{}{
		sender,
	})
	hw.Sum(senderAddressHash[:0])

	pub, err := crypto.Ecrecover(senderAddressHash[:], data.Proof[:])

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	if bytes.Compare(lockPublicKey, pub) != 0 {
		return Response{
			Code: code.CheckInvalidLock,
			Log:  fmt.Sprintf("Invalid proof")}
	}

	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)
	commission := big.NewInt(0).Set(commissionInBaseCoin)

	if decodedCheck.Coin != types.GetBaseCoin() {
		coin := context.GetStateCoin(decodedCheck.Coin)
		commission = formula.CalculateSaleAmount(coin.Volume(), coin.ReserveBalance(), coin.Data().Crr, commissionInBaseCoin)
	}

	totalTxCost := big.NewInt(0).Add(decodedCheck.Value, commission)

	if context.GetBalance(checkSender, decodedCheck.Coin).Cmp(totalTxCost) < 0 {
		return Response{
			Code: code.InsufficientFunds,
			Log:  fmt.Sprintf("Insufficient funds for check issuer account: %s. Wanted %d ", checkSender.String(), totalTxCost)}
	}

	if !isCheck {
		context.UseCheck(decodedCheck)
		rewardPull.Add(rewardPull, commissionInBaseCoin)

		if decodedCheck.Coin != types.GetBaseCoin() {
			context.SubCoinVolume(decodedCheck.Coin, commission)
			context.SubCoinReserve(decodedCheck.Coin, commissionInBaseCoin)
		}

		context.SubBalance(checkSender, decodedCheck.Coin, totalTxCost)
		context.AddBalance(sender, decodedCheck.Coin, decodedCheck.Value)
		context.SetNonce(sender, tx.Nonce)
	}

	tags := common.KVPairs{
		common.KVPair{Key: []byte("tx.type"), Value: []byte{TypeRedeemCheck}},
		common.KVPair{Key: []byte("tx.from"), Value: []byte(hex.EncodeToString(checkSender[:]))},
		common.KVPair{Key: []byte("tx.to"), Value: []byte(hex.EncodeToString(sender[:]))},
		common.KVPair{Key: []byte("tx.coin"), Value: []byte(decodedCheck.Coin.String())},
	}

	return Response{
		Code:      code.OK,
		Tags:      tags,
		GasUsed:   tx.Gas(),
		GasWanted: tx.Gas(),
	}
}
