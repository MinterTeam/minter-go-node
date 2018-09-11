package eventsdb

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
)

type Role string

var (
	RoleValidator  Role = "Validator"
	RoleDelegator  Role = "Delegator"
	RoleDAO        Role = "DAO"
	RoleDevelopers Role = "Developers"
)

type Event interface{}
type Events []Event

type RewardEvent struct {
	Role            Role
	Address         types.Address
	Amount          string
	ValidatorPubKey types.Pubkey
}

func (e RewardEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Role            string `json:"role"`
		Address         string `json:"address"`
		Amount          string `json:"amount"`
		ValidatorPubKey string `json:"validator_pub_key"`
	}{
		Role:            string(e.Role),
		Address:         e.Address.String(),
		Amount:          e.Amount,
		ValidatorPubKey: fmt.Sprintf("Mp%x", e.ValidatorPubKey),
	})
}

type SlashEvent struct {
	Address         types.Address
	Amount          string
	Coin            types.CoinSymbol
	ValidatorPubKey types.Pubkey
}

func (e SlashEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address         string `json:"address"`
		Amount          string `json:"amount"`
		Coin            string `json:"coin"`
		ValidatorPubKey string `json:"validator_pub_key"`
	}{
		Address:         e.Address.String(),
		Amount:          e.Amount,
		Coin:            e.Coin.String(),
		ValidatorPubKey: fmt.Sprintf("Mp%x", e.ValidatorPubKey),
	})
}
