package api

import (
	core_types "github.com/tendermint/tendermint/rpc/core/types"
)

func Genesis() (*core_types.ResultGenesis, error) {
	return client.Genesis()
}
