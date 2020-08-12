package waitlist

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/state/bus"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	"log"
	"math/big"
	"sort"
	"sync"
)

const mainPrefix = byte('w')

type RWaitList interface {
	Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) *Item
	GetByAddress(address types.Address) *Model
	GetByAddressAndPubKey(address types.Address, pubkey types.Pubkey) []Item
}

type WaitList struct {
	list  map[types.Address]*Model
	dirty map[types.Address]interface{}

	bus  *bus.Bus
	iavl tree.MTree

	lock sync.RWMutex
}

func NewWaitList(stateBus *bus.Bus, iavl tree.MTree) (*WaitList, error) {
	waitlist := &WaitList{
		bus:   stateBus,
		iavl:  iavl,
		list:  map[types.Address]*Model{},
		dirty: map[types.Address]interface{}{},
	}
	waitlist.bus.SetWaitList(NewBus(waitlist))

	return waitlist, nil
}

func (wl *WaitList) Commit() error {
	dirty := wl.getOrderedDirty()
	for _, address := range dirty {
		w := wl.getFromMap(address)

		wl.lock.Lock()
		delete(wl.dirty, address)
		wl.lock.Unlock()

		data, err := rlp.EncodeToBytes(w)
		if err != nil {
			return fmt.Errorf("can't encode object at %s: %v", address.String(), err)
		}

		path := append([]byte{mainPrefix}, address.Bytes()...)
		wl.iavl.Set(path, data)
	}

	return nil
}

func (wl *WaitList) GetByAddress(address types.Address) *Model {
	return wl.get(address)
}

func (wl *WaitList) Get(address types.Address, pubkey types.Pubkey, coin types.CoinID) *Item {
	waitlist := wl.get(address)
	if waitlist == nil {
		return nil
	}

	for _, item := range waitlist.List {
		if item.PublicKey == pubkey && item.Coin == coin {
			return &item
		}
	}

	return nil
}

func (wl *WaitList) GetByAddressAndPubKey(address types.Address, pubkey types.Pubkey) []Item {
	waitlist := wl.get(address)
	if waitlist == nil {
		return nil
	}

	items := make([]Item, 0, len(waitlist.List))
	for i, item := range waitlist.List {
		if item.PublicKey == pubkey {
			items[i] = item
		}
	}

	if len(items) == 0 {
		return nil
	}

	return items
}

func (wl *WaitList) AddWaitList(address types.Address, pubkey types.Pubkey, coin types.CoinID, value *big.Int) {
	w := wl.getOrNew(address)
	w.AddToList(pubkey, coin, value)
	wl.setToMap(address, w)
	w.markDirty(address)
}

func (wl *WaitList) Delete(address types.Address, pubkey types.Pubkey, coin types.CoinID) {
	w := wl.get(address)
	if w == nil || len(w.List) == 0 {
		log.Panicf("Waitlist not found for %s", address.String())
	}

	items := make([]Item, 0, len(w.List))
	for _, item := range w.List {
		if item.PublicKey != pubkey && item.Coin != coin {
			items = append(items, item)
		}
	}

	w.List = items
	wl.markDirty(address)
	wl.setToMap(address, w)
}

func (wl *WaitList) getOrNew(address types.Address) *Model {
	w := wl.get(address)
	if w == nil {
		w = &Model{List: make([]Item, 0), address: address, markDirty: wl.markDirty}
		wl.setToMap(address, w)
	}

	return w
}

func (wl *WaitList) get(address types.Address) *Model {
	if ff := wl.getFromMap(address); ff != nil {
		return ff
	}

	path := append([]byte{mainPrefix}, address.Bytes()...)
	_, enc := wl.iavl.Get(path)
	if len(enc) == 0 {
		return nil
	}

	m := new(Model)
	if err := rlp.DecodeBytes(enc, m); err != nil {
		panic(fmt.Sprintf("failed to decode waitlists for address %s: %s", address.String(), err))
	}

	m.address = address
	m.markDirty = wl.markDirty
	wl.setToMap(address, m)

	return m
}

func (wl *WaitList) getFromMap(address types.Address) *Model {
	wl.lock.RLock()
	defer wl.lock.RUnlock()

	return wl.list[address]
}

func (wl *WaitList) setToMap(address types.Address, model *Model) {
	wl.lock.Lock()
	defer wl.lock.Unlock()

	wl.list[address] = model
}

func (wl *WaitList) markDirty(address types.Address) {
	wl.dirty[address] = struct{}{}
}

func (wl *WaitList) getOrderedDirty() []types.Address {
	keys := make([]types.Address, 0, len(wl.dirty))
	for k := range wl.dirty {
		keys = append(keys, k)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return bytes.Compare(keys[i].Bytes(), keys[j].Bytes()) == 1
	})

	return keys
}
