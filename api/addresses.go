package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type AddressesResponse struct {
	Address          string        `json:"address"`
	Balance          []BalanceItem `json:"balance"`
	TransactionCount uint64        `json:"transaction_count"`
}

func Addresses(addresses []types.Address, height int) (*[]AddressesResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	response := make([]AddressesResponse, len(addresses))

	for i, address := range addresses {
		balances := cState.Accounts().GetBalances(address)

		data := AddressesResponse{
			Address:          address.String(),
			Balance:          make([]BalanceItem, len(balances)),
			TransactionCount: cState.Accounts().GetNonce(address),
		}

		isBaseCoinExists := false
		for k, b := range balances {
			data.Balance[k] = BalanceItem{
				CoinID: b.Coin.ID.Uint32(),
				Symbol: b.Coin.GetFullSymbol(),
				Value:  b.Value.String(),
			}

			if b.Coin.ID.IsBaseCoin() {
				isBaseCoinExists = true
			}
		}

		if !isBaseCoinExists {
			data.Balance = append(data.Balance, BalanceItem{
				CoinID: types.GetBaseCoinID().Uint32(),
				Symbol: types.GetBaseCoin().String(),
				Value:  "0",
			})
		}

		response[i] = data
	}

	return &response, nil
}
