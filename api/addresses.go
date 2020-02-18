package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type AddressesResponse struct {
	Address          string            `json:"address"`
	Balance          map[string]string `json:"balance"`
	TransactionCount uint64            `json:"transaction_count"`
}

func Addresses(addresses []types.Address, height int) (*[]AddressesResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.Lock()
	defer cState.Unlock()

	response := make([]AddressesResponse, len(addresses))

	for i, address := range addresses {
		data := AddressesResponse{
			Address:          address.String(),
			Balance:          make(map[string]string),
			TransactionCount: cState.Accounts.GetNonce(address),
		}

		balances := cState.Accounts.GetBalances(address)
		for k, v := range balances {
			data.Balance[k.String()] = v.String()
		}

		if _, exists := data.Balance[types.GetBaseCoin().String()]; !exists {
			data.Balance[types.GetBaseCoin().String()] = "0"
		}

		response[i] = data
	}

	return &response, nil
}
