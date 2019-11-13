package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type AddressResponse struct {
	Balance          map[string]string `json:"balance"`
	TransactionCount uint64            `json:"transaction_count"`
}

func Address(address types.Address, height int) (*AddressResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	response := AddressResponse{
		Balance:          make(map[string]string),
		TransactionCount: cState.Accounts.GetNonce(address),
	}

	balances := cState.Accounts.GetBalances(address)

	for k, v := range balances {
		response.Balance[k.String()] = v.String()
	}

	if _, exists := response.Balance[types.GetBaseCoin().String()]; !exists {
		response.Balance[types.GetBaseCoin().String()] = "0"
	}

	return &response, nil
}
