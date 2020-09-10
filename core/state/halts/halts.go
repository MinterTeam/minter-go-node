package halts

import (
	"encoding/binary"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"sort"
	"sync"
)

const mainPrefix = byte('h')

type RHalts interface {
	Export(state *types.AppState)
	GetHaltBlocks(height uint64) *Model
}

type HaltBlocks struct {
	list  map[uint64]*Model
	dirty map[uint64]interface{}

	bus  *bus.Bus
	iavl tree.MTree

	lock sync.RWMutex
}

func NewHalts(stateBus *bus.Bus, iavl tree.MTree) (*HaltBlocks, error) {
	halts := &HaltBlocks{
		bus:   stateBus,
		iavl:  iavl,
		list:  map[uint64]*Model{},
		dirty: map[uint64]interface{}{},
	}

	halts.bus.SetHaltBlocks(NewBus(halts))

	return halts, nil
}

func (hb *HaltBlocks) Commit() error {
	dirty := hb.getOrderedDirty()
	for _, height := range dirty {
		haltBlock := hb.getFromMap(height)

		hb.lock.Lock()
		delete(hb.dirty, height)
		hb.lock.Unlock()

		path := getPath(height)

		if haltBlock.deleted {
			hb.lock.Lock()
			delete(hb.list, height)
			hb.lock.Unlock()

			hb.iavl.Remove(path)
		} else {
			data, err := rlp.EncodeToBytes(haltBlock)
			if err != nil {
				return fmt.Errorf("can't encode object at %d: %v", height, err)
			}

			hb.iavl.Set(path, data)
		}
	}

	return nil
}

func (hb *HaltBlocks) GetHaltBlocks(height uint64) *Model {
	return hb.get(height)
}

func (hb *HaltBlocks) GetOrNew(height uint64) *Model {
	haltBlock := hb.get(height)
	if haltBlock == nil {
		haltBlock = &Model{
			height:    height,
			markDirty: hb.markDirty,
		}
		hb.setToMap(height, haltBlock)
	}

	return haltBlock
}

func (hb *HaltBlocks) get(height uint64) *Model {
	if haltBlock := hb.getFromMap(height); haltBlock != nil {
		return haltBlock
	}

	_, enc := hb.iavl.Get(getPath(height))
	if len(enc) == 0 {
		return nil
	}

	haltBlock := &Model{}
	if err := rlp.DecodeBytes(enc, haltBlock); err != nil {
		panic(fmt.Sprintf("failed to decode halt blocks at height %d: %s", height, err))
	}

	haltBlock.height = height
	haltBlock.markDirty = hb.markDirty

	hb.setToMap(height, haltBlock)

	return haltBlock
}

func (hb *HaltBlocks) markDirty(height uint64) {
	hb.dirty[height] = struct{}{}
}

func (hb *HaltBlocks) getOrderedDirty() []uint64 {
	keys := make([]uint64, 0, len(hb.dirty))
	for k := range hb.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func (hb *HaltBlocks) AddHaltBlock(height uint64, pubkey types.Pubkey) {
	hb.GetOrNew(height).addHaltBlock(pubkey)
}

func (hb *HaltBlocks) Delete(height uint64) {
	haltBlock := hb.get(height)
	if haltBlock == nil {
		return
	}

	haltBlock.delete()
}

func (hb *HaltBlocks) Export(state *types.AppState) {
	hb.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] != mainPrefix {
			return false
		}

		height := binary.LittleEndian.Uint64(key[1:])
		halts := hb.get(height)
		if halts == nil {
			return false
		}

		for _, haltBlock := range halts.List {
			state.HaltBlocks = append(state.HaltBlocks, types.HaltBlock{
				Height:       height,
				CandidateKey: haltBlock.Pubkey,
			})
		}

		return false
	})
}

func (hb *HaltBlocks) getFromMap(height uint64) *Model {
	hb.lock.RLock()
	defer hb.lock.RUnlock()

	return hb.list[height]
}

func (hb *HaltBlocks) setToMap(height uint64, model *Model) {
	hb.lock.Lock()
	defer hb.lock.Unlock()

	hb.list[height] = model
}

func getPath(height uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, height)

	return append([]byte{mainPrefix}, b...)
}
