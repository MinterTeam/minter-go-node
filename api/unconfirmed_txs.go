package api

import (
	"github.com/tendermint/tendermint/rpc/core/types"
)

func UnconfirmedTxs(limit int) (*core_types.ResultUnconfirmedTxs, error) {
	return client.UnconfirmedTxs(limit)
}
