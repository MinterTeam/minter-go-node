package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type AddressesResponse struct {
	Address          types.Address       `json:"address"`
	Balance          map[string]*big.Int `json:"balance"`
	TransactionCount uint64              `json:"transaction_count"`
}

func Addresses(addresses []types.Address, height int) (*[]AddressesResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	response := make([]AddressesResponse, len(addresses))

	for i, address := range addresses {
		data := AddressesResponse{
			Address:          address,
			Balance:          make(map[string]*big.Int),
			TransactionCount: cState.GetNonce(address),
		}

		balances := cState.GetBalances(address)
		for k, v := range balances.Data {
			data.Balance[k.String()] = v
		}

		if _, exists := data.Balance[types.GetBaseCoin().String()]; !exists {
			data.Balance[types.GetBaseCoin().String()] = big.NewInt(0)
		}

		response[i] = data
	}

	return &response, nil
}
