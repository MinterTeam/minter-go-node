package frozenfunds

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"math/big"
	"sync"
)

type Item struct {
	Address         types.Address
	CandidateKey    *types.Pubkey
	CandidateID     uint32
	Coin            types.CoinID
	Value           *big.Int
	MoveToCandidate *uint32 `rlp:"nilList"`
}

type Model struct {
	List []Item

	height    uint64
	deleted   bool
	markDirty func(height uint64)
	lock      sync.RWMutex
}

func (m *Model) delete() {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.deleted = true
	m.markDirty(m.height)
}

func (m *Model) addFund(address types.Address, pubkey types.Pubkey, candidateID uint32, coin types.CoinID, value *big.Int, moveToCandidateID *uint32) {
	m.lock.Lock()
	m.List = append(m.List, Item{
		Address:         address,
		CandidateKey:    &pubkey,
		CandidateID:     candidateID,
		Coin:            coin,
		Value:           value,
		MoveToCandidate: moveToCandidateID,
	})
	m.lock.Unlock()

	m.markDirty(m.height)
}

func (m *Model) Height() uint64 {
	return m.height
}
