package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type SendData struct {
	Coin  types.CoinSymbol
	To    types.Address
	Value *big.Int
}

func (s SendData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Coin  types.CoinSymbol `json:"coin,string"`
		To    types.Address    `json:"to"`
		Value string           `json:"value"`
	}{
		Coin:  s.Coin,
		To:    s.To,
		Value: s.Value.String(),
	})
}

func (s SendData) String() string {
	return fmt.Sprintf("SEND to:%s coin:%s value:%s",
		s.To.String(), s.Coin.String(), s.Value.String())
}

func (s SendData) Gas() int64 {
	return commissions.SendTx
}
