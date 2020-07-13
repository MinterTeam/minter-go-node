package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type AddressesResponse struct {
	Address          string            `json:"address"`
	Balance          []BalanceItem     `json:"balances"`
	TransactionCount uint64            `json:"transaction_count"`
}

func Addresses(addresses []types.Address, height int) (*[]AddressesResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	response := make([]AddressesResponse, len(addresses))

	for i, address := range addresses {
		balances := cState.Accounts().GetBalances(address)

		data := AddressesResponse{
			Address:          address.String(),
			Balance:          make([]BalanceItem, len(balances)),
			TransactionCount: cState.Accounts().GetNonce(address),
		}

		for k, b := range balances {
			data.Balance[k] = BalanceItem{
				CoinID: b.Coin.ID.Uint32(),
				Symbol: b.Coin.GetFullSymbol(),
				Value:  b.Value.String(),
			}
		}

		//if _, exists := data.Balance[types.GetBaseCoin().String()]; !exists {
		//	data.Balance[types.GetBaseCoin().String()] = "0"
		//}

		response[i] = data
	}

	return &response, nil
}
