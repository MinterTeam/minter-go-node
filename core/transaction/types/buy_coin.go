package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type BuyCoinData struct {
	CoinToBuy  types.CoinSymbol
	ValueToBuy *big.Int
	CoinToSell types.CoinSymbol
}

func (s BuyCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToBuy  types.CoinSymbol `json:"coin_to_buy,string"`
		ValueToBuy string           `json:"value_to_buy"`
		CoinToSell types.CoinSymbol `json:"coin_to_sell,string"`
	}{
		CoinToBuy:  s.CoinToBuy,
		ValueToBuy: s.ValueToBuy.String(),
		CoinToSell: s.CoinToSell,
	})
}

func (s BuyCoinData) String() string {
	return fmt.Sprintf("BUY COIN sell:%s buy:%s %s",
		s.CoinToSell.String(), s.ValueToBuy.String(), s.CoinToBuy.String())
}

func (s BuyCoinData) Gas() int64 {
	return commissions.ConvertTx
}
