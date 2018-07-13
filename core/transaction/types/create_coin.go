package types

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type CreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint
}

func (s CreateCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name                 string           `json:"name"`
		Symbol               types.CoinSymbol `json:"coin_symbol"`
		InitialAmount        string           `json:"initial_amount"`
		InitialReserve       string           `json:"initial_reserve"`
		ConstantReserveRatio uint             `json:"constant_reserve_ratio"`
	}{
		Name:                 s.Name,
		Symbol:               s.Symbol,
		InitialAmount:        s.InitialAmount.String(),
		InitialReserve:       s.InitialReserve.String(),
		ConstantReserveRatio: s.ConstantReserveRatio,
	})
}

func (s CreateCoinData) String() string {
	return fmt.Sprintf("CREATE COIN symbol:%s reserve:%s amount:%s crr:%d",
		s.Symbol.String(), s.InitialReserve, s.InitialAmount, s.ConstantReserveRatio)
}

func (s CreateCoinData) Gas() int64 {
	return commissions.CreateTx
}
