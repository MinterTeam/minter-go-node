package accounts

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"math/big"
)

type Accounts struct {
	db db.DB
}

func NewAccounts(db db.DB) (*Accounts, error) {
	return &Accounts{db: db}, nil
}

func (v *Accounts) Commit() error {
	panic("implement me")
}

func (v *Accounts) AddBalance(address types.Address, coin types.CoinSymbol, amount *big.Int) {
	panic("implement me")
}
