package api

import (
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
)

func MaxGas(height int) (*uint64, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "State at given height not found", Data: err.Error()}
	}

	maxGas := cState.GetCurrentMaxGas()
	return &maxGas, nil
}
