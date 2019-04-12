package events

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

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
