package eventsdb

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Role byte

func (r Role) String() string {
	switch r {
	case RoleValidator:
		return "Validator"
	case RoleDelegator:
		return "Delegator"
	case RoleDAO:
		return "DAO"
	case RoleDevelopers:
		return "Developers"
	}

	return "Undefined"
}

const (
	RoleValidator Role = iota
	RoleDelegator
	RoleDAO
	RoleDevelopers
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
		Role            string       `json:"role"`
		Address         string       `json:"address"`
		Amount          string       `json:"amount"`
		ValidatorPubKey types.Pubkey `json:"validator_pub_key"`
	}{
		Role:            e.Role.String(),
		Address:         e.Address.String(),
		Amount:          big.NewInt(0).SetBytes(e.Amount).String(),
		ValidatorPubKey: e.ValidatorPubKey,
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
		Address         string       `json:"address"`
		Amount          string       `json:"amount"`
		Coin            string       `json:"coin"`
		ValidatorPubKey types.Pubkey `json:"validator_pub_key"`
	}{
		Address:         e.Address.String(),
		Amount:          big.NewInt(0).SetBytes(e.Amount).String(),
		Coin:            e.Coin.String(),
		ValidatorPubKey: e.ValidatorPubKey,
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
		Address         string       `json:"address"`
		Amount          string       `json:"amount"`
		Coin            string       `json:"coin"`
		ValidatorPubKey types.Pubkey `json:"validator_pub_key"`
	}{
		Address:         e.Address.String(),
		Amount:          big.NewInt(0).SetBytes(e.Amount).String(),
		Coin:            e.Coin.String(),
		ValidatorPubKey: e.ValidatorPubKey,
	})
}

type CoinLiquidationEvent struct {
	Coin types.CoinSymbol
}

func (e CoinLiquidationEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Coin string `json:"coin"`
	}{
		Coin: e.Coin.String(),
	})
}
