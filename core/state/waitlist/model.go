package waitlist

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
)

type Item struct {
	CandidateId uint32
	Coin        types.CoinID
	Value       *big.Int
}

type Model struct {
	List []Item

	address   types.Address
	markDirty func(address types.Address)
}

func (m *Model) AddToList(candidateId uint32, coin types.CoinID, value *big.Int) {
	m.List = append(m.List, Item{
		CandidateId: candidateId,
		Coin:        coin,
		Value:       new(big.Int).Set(value),
	})
}
