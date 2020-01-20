package api

import (
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

func UnconfirmedTxs(limit int) (*core_types.ResultUnconfirmedTxs, error) {
	return client.UnconfirmedTxs(limit)
}
