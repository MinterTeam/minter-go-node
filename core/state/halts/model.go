package halts

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

type Item struct {
	Pubkey types.Pubkey
}

type Model struct {
	List []Item

	height    uint64
	deleted   bool
	markDirty func(height uint64)
}

func (m *Model) delete() {
	m.deleted = true
	m.markDirty(m.height)
}

func (m *Model) addHaltBlock(pubkey types.Pubkey) {
	m.List = append(m.List, Item{
		Pubkey: pubkey,
	})
	m.markDirty(m.height)
}

func (m *Model) Height() uint64 {
	return m.height
}
