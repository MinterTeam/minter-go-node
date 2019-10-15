package frozen_funds

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Item struct {
	Address      types.Address
	CandidateKey *types.Pubkey
	Coin         types.CoinSymbol
	Value        *big.Int
}

type Model struct {
	List []Item

	height    uint64
	deleted   bool
	markDirty func(height uint64)
}

func (m *Model) Delete() {
	m.deleted = true
	m.markDirty(m.height)
}

func (m *Model) AddFund(address types.Address, pubkey types.Pubkey, coin types.CoinSymbol, value *big.Int) {
	m.List = append(m.List, Item{
		Address:      address,
		CandidateKey: &pubkey,
		Coin:         coin,
		Value:        value,
	})
	m.markDirty(m.height)
}
