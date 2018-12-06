package api

import (
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/rpc/core/types"
)

func UnconfirmedTxs(limit int) (*core_types.ResultUnconfirmedTxs, error) {
	return nil, errors.New("not implemented")
}
