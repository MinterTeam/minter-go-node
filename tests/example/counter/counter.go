package counter

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"

	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
)

type Application struct {
	types.BaseApplication

	hashCount int
	TxCount   int
	serial    bool

	txDecoder transaction.DecoderTx
}

func NewApplication(serial bool, decoder transaction.DecoderTx) *Application {
	return &Application{serial: serial, txDecoder: decoder}
}

func (app *Application) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{Data: fmt.Sprintf("{\"hashes\":%v,\"txs\":%v}", app.hashCount, app.TxCount)}
}

func (app *Application) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	key, value := req.Key, req.Value
	if key == "serial" && value == "on" {
		app.serial = true
	} else {
		/*
			TODO Panic and have the ABCI server pass an exception.
			The client can call SetOptionSync() and get an `error`.
			return types.ResponseSetOption{
				Error: fmt.Sprintf("Unknown key (%s) or value (%s)", key, value),
			}
		*/
		return types.ResponseSetOption{}
	}

	return types.ResponseSetOption{}
}

func (app *Application) DeliverTx(req types.RequestDeliverTx) types.ResponseDeliverTx {
	if app.serial {
		if len(req.Tx) > 2500 {
			return types.ResponseDeliverTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Max tx size is 2500 bytes, got %d", len(req.Tx))}
		}

		if app.txDecoder != nil {
			tx, _ := app.txDecoder.DecodeFromBytes(req.Tx)
			if tx.Nonce != uint64(app.TxCount) {
				return types.ResponseDeliverTx{
					Code: code.CodeTypeBadNonce,
					Log:  fmt.Sprintf("Invalid nonce. Expected %v, got %v", app.TxCount, tx.Nonce)}
			}
		}
	}
	app.TxCount++
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

func (app *Application) CheckTx(req types.RequestCheckTx) types.ResponseCheckTx {
	if app.serial {
		if len(req.Tx) > 2500 {
			return types.ResponseCheckTx{
				Code: code.CodeTypeEncodingError,
				Log:  fmt.Sprintf("Max tx size is 2500 bytes, got %d", len(req.Tx))}
		}

		if app.txDecoder != nil {
			tx, _ := app.txDecoder.DecodeFromBytes(req.Tx)
			if tx.Nonce != uint64(app.TxCount) {
				return types.ResponseCheckTx{
					Code: code.CodeTypeBadNonce,
					Log:  fmt.Sprintf("Invalid nonce. Expected >= %v, got %v", app.TxCount, tx.Nonce)}
			}
		}
	}
	return types.ResponseCheckTx{Code: code.CodeTypeOK}
}

func (app *Application) Commit() (resp types.ResponseCommit) {
	app.hashCount++
	if app.TxCount == 0 {
		return types.ResponseCommit{}
	}
	hash := make([]byte, 8)
	binary.BigEndian.PutUint64(hash, uint64(app.TxCount))
	return types.ResponseCommit{Data: hash}
}

func (app *Application) Query(reqQuery types.RequestQuery) types.ResponseQuery {
	switch reqQuery.Path {
	case "hash":
		return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.hashCount))}
	case "tx":
		return types.ResponseQuery{Value: []byte(fmt.Sprintf("%v", app.TxCount))}
	default:
		return types.ResponseQuery{Log: fmt.Sprintf("Invalid query path. Expected hash or tx, got %v", reqQuery.Path)}
	}
}
