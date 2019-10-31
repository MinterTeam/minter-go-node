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
	maxTxLength          = 7168
	maxPayloadLength     = 1024
	maxServiceDataLength = 128

	createCoinGas = 5000
)

type Response struct {
	Code      uint32          `json:"code,omitempty"`
	Data      []byte          `json:"data,omitempty"`
	Log       string          `json:"log,omitempty"`
	Info      string          `json:"info,omitempty"`
	GasWanted int64           `json:"gas_wanted,omitempty"`
	GasUsed   int64           `json:"gas_used,omitempty"`
	Tags      []common.KVPair `json:"tags,omitempty"`
	GasPrice  uint32          `json:"gas_price"`
}

func RunTx(context *state.State,
	isCheck bool,
	rawTx []byte,
	rewardPool *big.Int,
	currentBlock uint64,
	currentMempool sync.Map,
	minGasPrice uint32) Response {
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

	if tx.ChainID != types.CurrentChainID {
		return Response{
			Code: code.WrongChainID,
			Log:  "Wrong chain id"}
	}

	if !context.Coins.Exists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin)}
	}

	if isCheck && tx.GasPrice < minGasPrice {
		return Response{
			Code: code.TooLowGasPrice,
			Log:  fmt.Sprintf("Gas price of tx is too low to be included in mempool. Expected %d", minGasPrice),
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
		multisig := context.Accounts.GetAccount(tx.multisig.Multisig)

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

	if expectedNonce := context.Accounts.GetNonce(sender) + 1; expectedNonce != tx.Nonce {
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
		context.Coins.Sanitize(tx.GasCoin)
	}

	if tx.Type == TypeCreateCoin {
		response.GasUsed = createCoinGas
		response.GasWanted = createCoinGas
	}

	return response
}
