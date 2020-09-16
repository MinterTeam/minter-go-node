package transaction

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/MinterTeam/minter-go-node/core/code"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/kv"
)

var (
	CommissionMultiplier = big.NewInt(10e14)
)

const (
	maxTxLength          = 7168
	maxPayloadLength     = 1024
	maxServiceDataLength = 128

	stdGas = 5000
)

type Response struct {
	Code      uint32    `json:"code,omitempty"`
	Data      []byte    `json:"data,omitempty"`
	Log       string    `json:"log,omitempty"`
	Info      string    `json:"-"`
	GasWanted int64     `json:"gas_wanted,omitempty"`
	GasUsed   int64     `json:"gas_used,omitempty"`
	Tags      []kv.Pair `json:"tags,omitempty"`
	GasPrice  uint32    `json:"gas_price"`
}

func RunTx(context state.Interface,
	rawTx []byte,
	rewardPool *big.Int,
	currentBlock uint64,
	currentMempool *sync.Map,
	minGasPrice uint32) Response {
	lenRawTx := len(rawTx)
	if lenRawTx > maxTxLength {
		return Response{
			Code: code.TxTooLarge,
			Log:  fmt.Sprintf("TX length is over %d bytes", maxTxLength),
			Info: EncodeError(code.NewTxTooLarge(fmt.Sprintf("%d", maxTxLength), fmt.Sprintf("%d", lenRawTx))),
		}
	}

	tx, err := TxDecoder.DecodeFromBytes(rawTx)
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	if tx.ChainID != types.CurrentChainID {
		return Response{
			Code: code.WrongChainID,
			Log:  "Wrong chain id",
			Info: EncodeError(code.NewWrongChainID(fmt.Sprintf("%d", types.CurrentChainID), fmt.Sprintf("%d", tx.ChainID))),
		}
	}

	var checkState *state.CheckState
	var isCheck bool
	if checkState, isCheck = context.(*state.CheckState); !isCheck {
		checkState = state.NewCheckState(context.(*state.State))
	}

	if !checkState.Coins().Exists(tx.GasCoin) {
		return Response{
			Code: code.CoinNotExists,
			Log:  fmt.Sprintf("Coin %s not exists", tx.GasCoin),
			Info: EncodeError(code.NewCoinNotExists("", tx.GasCoin.String())),
		}
	}

	if isCheck && tx.GasPrice < minGasPrice {
		return Response{
			Code: code.TooLowGasPrice,
			Log:  fmt.Sprintf("Gas price of tx is too low to be included in mempool. Expected %d", minGasPrice),
			Info: EncodeError(code.NewTooLowGasPrice(fmt.Sprintf("%d", minGasPrice), fmt.Sprintf("%d", tx.GasPrice))),
		}
	}

	lenPayload := len(tx.Payload)
	if lenPayload > maxPayloadLength {
		return Response{
			Code: code.TxPayloadTooLarge,
			Log:  fmt.Sprintf("TX payload length is over %d bytes", maxPayloadLength),
			Info: EncodeError(code.NewTxPayloadTooLarge(fmt.Sprintf("%d", maxPayloadLength), fmt.Sprintf("%d", lenPayload))),
		}
	}

	lenServiceData := len(tx.ServiceData)
	if lenServiceData > maxServiceDataLength {
		return Response{
			Code: code.TxServiceDataTooLarge,
			Log:  fmt.Sprintf("TX service data length is over %d bytes", maxServiceDataLength),
			Info: EncodeError(code.NewTxServiceDataTooLarge(fmt.Sprintf("%d", maxServiceDataLength), fmt.Sprintf("%d", lenServiceData))),
		}
	}

	sender, err := tx.Sender()
	if err != nil {
		return Response{
			Code: code.DecodeError,
			Log:  err.Error(),
			Info: EncodeError(code.NewDecodeError()),
		}
	}

	// check if mempool already has transactions from this address
	if _, has := currentMempool.Load(sender); isCheck && has {
		return Response{
			Code: code.TxFromSenderAlreadyInMempool,
			Log:  fmt.Sprintf("Tx from %s already exists in mempool", sender.String()),
			Info: EncodeError(code.NewTxFromSenderAlreadyInMempool(sender.String(), strconv.Itoa(int(currentBlock)))),
		}
	}

	if isCheck {
		currentMempool.Store(sender, true)
	}

	// check multi-signature
	if tx.SignatureType == SigTypeMulti {
		multisig := checkState.Accounts().GetAccount(tx.multisig.Multisig)

		if !multisig.IsMultisig() {
			return Response{
				Code: code.MultisigNotExists,
				Log:  "Multisig does not exists",
				Info: EncodeError(code.NewMultisigNotExists(tx.multisig.Multisig.String())),
			}
		}

		multisigData := multisig.Multisig()

		if len(tx.multisig.Signatures) > 32 || len(multisigData.Weights) < len(tx.multisig.Signatures) {
			return Response{
				Code: code.IncorrectMultiSignature,
				Log:  "Incorrect multi-signature",
				Info: EncodeError(code.NewIncorrectMultiSignature()),
			}
		}

		txHash := tx.Hash()
		var totalWeight uint
		var usedAccounts = map[types.Address]bool{}

		for _, sig := range tx.multisig.Signatures {
			signer, err := RecoverPlain(txHash, sig.R, sig.S, sig.V)
			if err != nil {
				return Response{
					Code: code.IncorrectMultiSignature,
					Log:  "Incorrect multi-signature",
					Info: EncodeError(code.NewIncorrectMultiSignature()),
				}
			}

			if usedAccounts[signer] {
				return Response{
					Code: code.DuplicatedAddresses,
					Log:  "Duplicated multisig addresses",
					Info: EncodeError(code.NewDuplicatedAddresses(signer.String())),
				}
			}

			usedAccounts[signer] = true
			totalWeight += multisigData.GetWeight(signer)
		}

		if totalWeight < multisigData.Threshold {
			return Response{
				Code: code.NotEnoughMultisigVotes,
				Log:  fmt.Sprintf("Not enough multisig votes. Needed %d, has %d", multisigData.Threshold, totalWeight),
				Info: EncodeError(code.NewNotEnoughMultisigVotes(fmt.Sprintf("%d", multisigData.Threshold), fmt.Sprintf("%d", totalWeight))),
			}
		}

	}

	if expectedNonce := checkState.Accounts().GetNonce(sender) + 1; expectedNonce != tx.Nonce {
		return Response{
			Code: code.WrongNonce,
			Log:  fmt.Sprintf("Unexpected nonce. Expected: %d, got %d.", expectedNonce, tx.Nonce),
			Info: EncodeError(code.NewWrongNonce(fmt.Sprintf("%d", expectedNonce), fmt.Sprintf("%d", tx.Nonce))),
		}
	}

	response := tx.decodedData.Run(tx, context, rewardPool, currentBlock)

	if response.Code != code.TxFromSenderAlreadyInMempool && response.Code != code.OK {
		currentMempool.Delete(sender)
	}

	response.GasPrice = tx.GasPrice

	switch tx.Type {
	case TypeCreateCoin, TypeEditCoinOwner, TypeRecreateCoin, TypeEditCandidatePublicKey:
		response.GasUsed = stdGas
		response.GasWanted = stdGas
	}

	return response
}

func EncodeError(data interface{}) string {
	marshal, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(marshal)
}
