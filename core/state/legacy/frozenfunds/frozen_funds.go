package frozenfunds

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"sort"
	"sync"
)

const mainPrefix = byte('f')

type RFrozenFunds interface {
	Export(state *types.AppState, height uint64)
}

type FrozenFunds struct {
	list  map[uint64]*Model
	dirty map[uint64]interface{}

	bus  *bus.Bus
	iavl tree.MTree

	lock sync.RWMutex
}

func NewFrozenFunds(stateBus *bus.Bus, iavl tree.MTree) (*FrozenFunds, error) {
	frozenfunds := &FrozenFunds{bus: stateBus, iavl: iavl, list: map[uint64]*Model{}, dirty: map[uint64]interface{}{}}
	return frozenfunds, nil
}

func (f *FrozenFunds) get(height uint64) *Model {
	if ff := f.getFromMap(height); ff != nil {
		return ff
	}

	_, enc := f.iavl.Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	ff := &Model{}
	if err := rlp.DecodeBytes(enc, ff); err != nil {
		panic(fmt.Sprintf("failed to decode frozen funds at height %d: %s", height, err))
	}

	ff.height = height
	ff.markDirty = f.markDirty

	f.setToMap(height, ff)

	return ff
}

func (f *FrozenFunds) markDirty(height uint64) {
	f.dirty[height] = struct{}{}
}

func (f *FrozenFunds) getOrderedDirty() []uint64 {
	keys := make([]uint64, 0, len(f.dirty))
	for k := range f.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (f *FrozenFunds) Export(state *types.AppState, height uint64, coinsMap map[types.CoinSymbol]types.Coin) {
	for i := height; i <= height+candidates.UnbondPeriod; i++ {
		frozenFunds := f.get(i)
		if frozenFunds == nil {
			continue
		}

		for _, frozenFund := range frozenFunds.List {
			state.FrozenFunds = append(state.FrozenFunds, types.FrozenFund{
				Height:       i,
				Address:      frozenFund.Address,
				CandidateKey: frozenFund.CandidateKey,
				Coin:         coinsMap[frozenFund.Coin].ID,
				Value:        frozenFund.Value.String(),
			})
		}
	}
}

func (f *FrozenFunds) getFromMap(height uint64) *Model {
	f.lock.RLock()
	defer f.lock.RUnlock()

	return f.list[height]
}

func (f *FrozenFunds) setToMap(height uint64, model *Model) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
