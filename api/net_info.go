package api

import (
	"github.com/tendermint/tendermint/rpc/core/types"
)

func NetInfo() (*core_types.ResultNetInfo, error) {
	return client.NetInfo()
}
