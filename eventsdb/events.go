package eventsdb

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Role string

func (r Role) Marshal() ([]byte, error) {
	switch r {
	case RoleValidator:
		return []byte{1}, nil
	case RoleDelegator:
		return []byte{2}, nil
	case RoleDAO:
		return []byte{3}, nil
	case RoleDevelopers:
		return []byte{4}, nil
	}

	return nil, fmt.Errorf("undefined role")
}

func (r *Role) Unmarshal(b []byte) error {
	switch b[0] {
	case 1:
		*r = RoleValidator
		return nil
	case 2:
		*r = RoleDelegator
		return nil
	case 3:
		*r = RoleDAO
		return nil
	case 4:
		*r = RoleDevelopers
		return nil
	}

	return fmt.Errorf("undefined role")
}

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
	Amount          []byte
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
		Amount:          big.NewInt(0).SetBytes(e.Amount).String(),
		ValidatorPubKey: fmt.Sprintf("Mp%x", e.ValidatorPubKey),
	})
}

type SlashEvent struct {
	Address         types.Address
	Amount          []byte
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
		Amount:          big.NewInt(0).SetBytes(e.Amount).String(),
		Coin:            e.Coin.String(),
		ValidatorPubKey: fmt.Sprintf("Mp%x", e.ValidatorPubKey),
	})
}

type UnbondEvent struct {
	Address         types.Address
	Amount          []byte
	Coin            types.CoinSymbol
	ValidatorPubKey types.Pubkey
}

func (e UnbondEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address         string `json:"address"`
		Amount          string `json:"amount"`
		Coin            string `json:"coin"`
		ValidatorPubKey string `json:"validator_pub_key"`
	}{
		Address:         e.Address.String(),
		Amount:          big.NewInt(0).SetBytes(e.Amount).String(),
		Coin:            e.Coin.String(),
		ValidatorPubKey: fmt.Sprintf("Mp%x", e.ValidatorPubKey),
	})
}
