package api

import (
	"github.com/tendermint/tendermint/rpc/core/types"
)

func SendTransaction(tx []byte) (*core_types.ResultBroadcastTx, error) {
	return client.BroadcastTxSync(tx)
}
