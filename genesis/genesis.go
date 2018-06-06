package genesis

import "minter/core/types"

type AppState struct {
	FirstValidatorAddress types.Address `json:"first_validator_address"`
	InitialBalances       []Account     `json:"initial_balances"`
}

type Account struct {
	Address types.Address               `json:"address"`
	Balance map[types.CoinSymbol]string `json:"balance"`
}
