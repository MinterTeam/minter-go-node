package state

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

var (
	BalanceChangeChan = make(chan BalanceChangeStruct, 10)
)

type BalanceChangeStruct struct {
	Address           types.Address
	Coin              types.CoinSymbol
	Balance           *big.Int
	BalanceInBasecoin *big.Int
}

func (s BalanceChangeStruct) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address           types.Address `json:"address"`
		Coin              string        `json:"coin"`
		Balance           string        `json:"balance"`
		BalanceInBasecoin string        `json:"balance_in_basecoin"`
	}{
		Address:           s.Address,
		Coin:              s.Coin.String(),
		Balance:           s.Balance.String(),
		BalanceInBasecoin: s.BalanceInBasecoin.String(),
	})
}

func EmitBalanceChange(address types.Address, coin types.CoinSymbol, balance *big.Int) {
	BalanceChangeChan <- BalanceChangeStruct{
		Address: address,
		Coin:    coin,
		Balance: balance,
	}
}
