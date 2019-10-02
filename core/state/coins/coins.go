package coins

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"math/big"
)

type Coins struct {
	db db.DB
}

func NewCoins(db db.DB) (*Coins, error) {
	return &Coins{db: db}, nil
}

func (v *Coins) Commit() error {
	panic("implement me")
}

func (v *Coins) GetCoin(symbol types.CoinSymbol) *Coin {
	panic("implement me")
}

func (v *Coins) Exists(symbol types.CoinSymbol) bool {
	panic("implement me")
}

func (v *Coins) SubVolume(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (v *Coins) AddVolume(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (v *Coins) SubReserve(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (v *Coins) AddReserve(symbol types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}

func (v *Coins) Sanitize(symbol types.CoinSymbol) {
	panic("implement me")
}

func (v *Coins) Create(symbol types.CoinSymbol, name string, volume *big.Int, crr uint, reserve *big.Int) {
	panic("implement me")
}

type Coin struct {
	Name           string
	Symbol         types.CoinSymbol
	Volume         *big.Int
	Crr            uint
	ReserveBalance *big.Int
}
