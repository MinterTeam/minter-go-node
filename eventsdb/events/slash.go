package events

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

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
