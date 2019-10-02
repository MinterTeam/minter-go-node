package frozen_funds

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	db "github.com/tendermint/tm-db"
	"math/big"
)

type FrozenFunds struct {
	db db.DB
}

func NewFrozenFunds(db db.DB) (*FrozenFunds, error) {
	return &FrozenFunds{db: db}, nil
}

func (v *FrozenFunds) Commit() error {
	panic("implement me")
}

func (v *FrozenFunds) GetFrozenFunds(height uint64) *FrozenFundsAtHeight {
	panic("implement me")
}

func (v *FrozenFunds) PunishFrozenFundsWithAddress(fromHeight uint64, toHeight interface{}, tmAddress [20]byte) {
	panic("implement me")
}

type FrozenFund struct {
	Address      types.Address
	CandidateKey []byte
	Coin         types.CoinSymbol
	Value        *big.Int
}

type FrozenFundsAtHeight struct {
	List []FrozenFund
}

func (ff *FrozenFundsAtHeight) Delete() {

}
