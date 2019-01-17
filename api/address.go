package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"math/big"
)

type AddressResponse struct {
	Balance          map[string]*big.Int `json:"balance"`
	TransactionCount uint64              `json:"transaction_count"`
}

func Address(address types.Address, height int) (*AddressResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "State at given height not found", Data: err.Error()}
	}

	response := AddressResponse{
		Balance:          make(map[string]*big.Int),
		TransactionCount: cState.GetNonce(address),
	}

	balances := cState.GetBalances(address)

	for k, v := range balances.Data {
		response.Balance[k.String()] = v
	}

	if _, exists := response.Balance[types.GetBaseCoin().String()]; !exists {
		response.Balance[types.GetBaseCoin().String()] = big.NewInt(0)
	}

	return &response, nil
}
