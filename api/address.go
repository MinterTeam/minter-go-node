package api

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type AddressResponse struct {
	Balance          []BalanceItem `json:"balances"`
	TransactionCount uint64        `json:"transaction_count"`
}

type BalanceItem struct {
	CoinID uint32 `json:"coin_id"`
	Symbol string `json:"symbol"`
	Value  string `json:"value"`
}

type Coin struct {
	ID     uint32 `json:"id"`
	Symbol string `json:"symbol"`
}

func Address(address types.Address, height int) (*AddressResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	balances := cState.Accounts().GetBalances(address)

	response := AddressResponse{
		Balance:          make([]BalanceItem, len(balances)),
		TransactionCount: cState.Accounts().GetNonce(address),
	}

	isBaseCoinExists := false
	for k, b := range balances {
		response.Balance[k] = BalanceItem{
			CoinID: b.Coin.ID.Uint32(),
			Symbol: b.Coin.GetFullSymbol(),
			Value:  b.Value.String(),
		}

		if b.Coin.ID.IsBaseCoin() {
			isBaseCoinExists = true
		}
	}

	if !isBaseCoinExists {
		response.Balance = append(response.Balance, BalanceItem{
			CoinID: types.GetBaseCoinID().Uint32(),
			Symbol: types.GetBaseCoin().String(),
			Value:  "0",
		})
	}

	return &response, nil
}
