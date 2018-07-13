package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type SellCoinData struct {
	CoinToSell  types.CoinSymbol
	ValueToSell *big.Int
	CoinToBuy   types.CoinSymbol
}

func (s SellCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CoinToSell  types.CoinSymbol `json:"coin_to_sell,string"`
		ValueToSell string           `json:"value_to_sell"`
		CoinToBuy   types.CoinSymbol `json:"coin_to_buy,string"`
	}{
		CoinToSell:  s.CoinToSell,
		ValueToSell: s.ValueToSell.String(),
		CoinToBuy:   s.CoinToBuy,
	})
}

func (s SellCoinData) String() string {
	return fmt.Sprintf("SELL COIN sell:%s %s buy:%s",
		s.ValueToSell.String(), s.CoinToBuy.String(), s.CoinToSell.String())
}

func (s SellCoinData) Gas() int64 {
	return commissions.ConvertTx
}
