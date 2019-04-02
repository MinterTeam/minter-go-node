package api

import (
	"github.com/tendermint/tendermint/rpc/core/types"
)

func Genesis() (*core_types.ResultGenesis, error) {
	return client.Genesis()
}
