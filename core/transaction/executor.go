package transaction

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/tendermint/tendermint/libs/common"
	"math/big"
	"sync"
)

var (
	CommissionMultiplier = big.NewInt(10e14)
)

const (
	maxTxLength          = maxPayloadLength + maxServiceDataLength + (1024 * 4) // TODO: make some estimations
	maxPayloadLength     = 1024
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
	GasPrice  *big.Int
}

func RunTx(context *state.StateDB, isCheck bool, rawTx []byte, rewardPool *big.Int, currentBlock uint64, currentMempool sync.Map, minGasPrice *big.Int) Response {
	if len(rawTx) > maxTxLength {
		return Response{
			Code: code.TxTooLarge,
			Log:  fmt.Sprintf("TX length is over %d bytes", maxTxLength)}
	}

	tx, err := TxDecoder.DecodeFromBytes(rawTx)
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	if isCheck && tx.GasPrice.Cmp(minGasPrice) == -1 {
		return Response{
			Code: code.TooLowGasPrice,
			Log:  fmt.Sprintf("Gas price of tx is too low to be included in mempool. Expected %s", minGasPrice),
		}
	}

	if !isCheck {
		log.Info("Deliver tx", "tx", tx.String())
	}

	if len(tx.Payload) > maxPayloadLength {
		return Response{
			Code: code.TxPayloadTooLarge,
			Log:  fmt.Sprintf("TX payload length is over %d bytes", maxPayloadLength)}
	}

	if len(tx.ServiceData) > maxServiceDataLength {
		return Response{
			Code: code.TxServiceDataTooLarge,
			Log:  fmt.Sprintf("TX service data length is over %d bytes", maxServiceDataLength)}
	}

	sender, err := tx.Sender()
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error()}
	}

	// check if mempool already has transactions from this address
	if _, has := currentMempool.Load(sender); isCheck && has {
		return Response{
			Code: code.TxFromSenderAlreadyInMempool,
			Log:  fmt.Sprintf("Tx from %s already exists in mempool", sender.String())}
	}

	if isCheck {
		currentMempool.Store(sender, true)
	}

	// check multi-signature
	if tx.SignatureType == SigTypeMulti {
		multisig := context.GetOrNewStateObject(tx.multisig.Multisig)

		if !multisig.IsMultisig() {
			return Response{
				Code: code.MultisigNotExists,
				Log:  "Multisig does not exists"}
		}

		multisigData := multisig.Multisig()

		if len(tx.multisig.Signatures) > 32 || len(multisigData.Weights) < len(tx.multisig.Signatures) {
			return Response{
				Code: code.IncorrectMultiSignature,
				Log:  "Incorrect multi-signature"}
		}

		txHash := tx.Hash()
		var totalWeight uint
		var usedAccounts = map[types.Address]bool{}

		for _, sig := range tx.multisig.Signatures {
			signer, err := RecoverPlain(txHash, sig.R, sig.S, sig.V)

			if err != nil {
				return Response{
					Code: code.IncorrectMultiSignature,
					Log:  "Incorrect multi-signature"}
			}

			if usedAccounts[signer] {
				return Response{
					Code: code.IncorrectMultiSignature,
					Log:  "Incorrect multi-signature"}
			}

			usedAccounts[signer] = true
			totalWeight += multisigData.GetWeight(signer)
		}

		if totalWeight < multisigData.Threshold {
			return Response{
				Code: code.IncorrectMultiSignature,
				Log:  fmt.Sprintf("Not enough multisig votes. Needed %d, has %d", multisigData.Threshold, totalWeight)}
		}
	}

	if expectedNonce := context.GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce)}
	}

	response := tx.decodedData.Run(tx, context, isCheck, rewardPool, currentBlock)

	if response.Code != code.TxFromSenderAlreadyInMempool && response.Code != code.OK {
		currentMempool.Delete(sender)
	}

	response.GasPrice = tx.GasPrice

	if !isCheck && response.Code == code.OK {
		context.DeleteCoinIfZeroReserve(tx.GasCoin)
	}

	return response
}
