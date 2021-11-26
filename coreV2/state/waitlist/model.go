package waitlist

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"sync"
)

type Item struct {
	CandidateId uint32
	Coin        types.CoinID
	Value       *big.Int
}

type Model struct {
	List []*Item

	address   types.Address
	markDirty func(address types.Address)
	lock      sync.RWMutex
}

func (m *Model) AddToList(candidateId uint32, coin types.CoinID, value *big.Int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.List = append(m.List, &Item{
		CandidateId: candidateId,
		Coin:        coin,
		Value:       new(big.Int).Set(value),
	})
}
