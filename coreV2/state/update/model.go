package update

import (
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"sync"
)

type Model struct {
	Votes   []types.Pubkey
	Version string

	height    uint64
	markDirty func()

	lock sync.Mutex
}

func (m *Model) addVote(pubkey types.Pubkey) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Votes = append(m.Votes, pubkey)
	m.markDirty()
}
