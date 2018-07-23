package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
)

var (
	CommissionMultiplier = big.NewInt(10e14)
)

const (
	maxTxLength          = 1024
	maxPayloadLength     = 128
	maxServiceDataLength = 128
)

type Response struct {
	Code      uint32          `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Data      []byte          `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Log       string          `protobuf:"bytes,3,opt,name=log,proto3" json:"log,omitempty"`
	Info      string          `protobuf:"bytes,4,opt,name=info,proto3" json:"info,omitempty"`
	GasWanted int64           `protobuf:"varint,5,opt,name=gas_wanted,json=gasWanted,proto3" json:"gas_wanted,omitempty"`
	GasUsed   int64           `protobuf:"varint,6,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	Tags      []common.KVPair `protobuf:"bytes,7,rep,name=tags" json:"tags,omitempty"`
	Fee       common.KI64Pair `protobuf:"bytes,8,opt,name=fee" json:"fee"`
}

func RunTx(context *state.StateDB, isCheck bool, rawTx []byte, rewardPull *big.Int, currentBlock uint64) Response {

	if len(rawTx) > maxTxLength {
		return Response{
			Code: code.TxTooLarge,
			Log:  "TX length is over 1024 bytes"}
	}

	tx, err := DecodeFromBytes(rawTx)

	if !isCheck {
		log.Info("Deliver tx", "tx", tx.String())
	}

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	if len(tx.Payload) > maxPayloadLength {
		return Response{
			Code: code.TxPayloadTooLarge,
			Log:  "TX payload length is over 128 bytes"}
	}

	if len(tx.ServiceData) > maxServiceDataLength {
		return Response{
			Code: code.TxServiceDataTooLarge,
			Log:  "TX service data length is over 128 bytes"}
	}

	sender, err := tx.Sender()

	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	// TODO: deal with multiple pending transactions from one account
	if expectedNonce := context.GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce)}
	}

	return tx.decodedData.Run(sender, tx, context, isCheck, rewardPull, currentBlock)
}
