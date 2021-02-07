package events

import (
	"encoding/binary"
	tmjson "github.com/tendermint/tendermint/libs/json"
	db "github.com/tendermint/tm-db"
	"sync"
)

func init() {
	tmjson.RegisterType(&reward{}, "reward")
	tmjson.RegisterType(&slash{}, "slash")
	tmjson.RegisterType(&unbond{}, "unbond")
	tmjson.RegisterType(&kick{}, "kick")
	tmjson.RegisterType(&move{}, "move")
	tmjson.RegisterType(&RewardEvent{}, TypeRewardEvent)
	tmjson.RegisterType(&SlashEvent{}, TypeSlashEvent)
	tmjson.RegisterType(&UnbondEvent{}, TypeUnbondEvent)
	tmjson.RegisterType(&StakeKickEvent{}, TypeStakeKickEvent)
	tmjson.RegisterType(&StakeMoveEvent{}, TypeStakeMoveEvent)
	tmjson.RegisterType(&UpdateNetworkEvent{}, TypeUpdateNetworkEvent)
	tmjson.RegisterType(&UpdateCommissionsEvent{}, TypeUpdateCommissionsEvent)
}

// IEventsDB is an interface of Events
type IEventsDB interface {
	AddEvent(height uint32, event Event)
	LoadEvents(height uint32) Events
	CommitEvents() error
	Close() error
}

type MockEvents struct{}

func (e MockEvents) AddEvent(height uint32, event Event) {}
func (e MockEvents) LoadEvents(height uint32) Events     { return Events{} }
func (e MockEvents) CommitEvents() error                 { return nil }
func (e MockEvents) Close() error                        { return nil }

type eventsStore struct {
	sync.RWMutex
	db        db.DB
	pending   pendingEvents
	idPubKey  map[uint16][32]byte
	pubKeyID  map[[32]byte]uint16
	idAddress map[uint32][20]byte
	addressID map[[20]byte]uint32
}

type pendingEvents struct {
	sync.Mutex
	height uint32
	items  Events
}

// NewEventsStore creates new events store in given DB
func NewEventsStore(db db.DB) IEventsDB {
	return &eventsStore{
		RWMutex:   sync.RWMutex{},
		db:        db,
		pending:   pendingEvents{},
		idPubKey:  make(map[uint16][32]byte),
		pubKeyID:  make(map[[32]byte]uint16),
		idAddress: make(map[uint32][20]byte),
		addressID: make(map[[20]byte]uint32),
	}
}

func (store *eventsStore) cachePubKey(id uint16, key [32]byte) {
	store.idPubKey[id] = key
	store.pubKeyID[key] = id
}

func (store *eventsStore) Close() error {
	return store.db.Close()
}

func (store *eventsStore) cacheAddress(id uint32, address [20]byte) {
	store.idAddress[id] = address
	store.addressID[address] = id
}

func (store *eventsStore) AddEvent(height uint32, event Event) {
	store.pending.Lock()
	defer store.pending.Unlock()
	if store.pending.height != height {
		store.pending.items = Events{}
	}
	store.pending.items = append(store.pending.items, event)
	store.pending.height = height
}

func (store *eventsStore) LoadEvents(height uint32) Events {
	store.loadCache()

	ok, err := store.db.Has(uint32ToBytes(height))
	if !ok {
		return nil
	}

	bytes, err := store.db.Get(uint32ToBytes(height))
	if err != nil {
		panic(err)
	}
	if len(bytes) == 0 {
		return Events{}
	}

	var items []compact
	if err := tmjson.Unmarshal(bytes, &items); err != nil {
		panic(err)
	}

	resultEvents := make(Events, 0, len(items))
	for _, compactEvent := range items {
		if stake, ok := compactEvent.(stake); ok {
			resultEvents = append(resultEvents, stake.compile(store.idPubKey[stake.pubKeyID()], store.idAddress[stake.addressID()]))
		} else if c, ok := compactEvent.(Event); ok {
			resultEvents = append(resultEvents, c)
		} else {
			panic("unemployment event interface")
		}
	}

	return resultEvents
}

func (store *eventsStore) CommitEvents() error {
	store.loadCache()

	store.pending.Lock()
	defer store.pending.Unlock()
	var data []compact
	for _, item := range store.pending.items {
		if stake, ok := item.(Stake); ok {
			pubKey := store.savePubKey(stake.validatorPubKey())
			address := store.saveAddress(stake.address())
			data = append(data, stake.convert(pubKey, address))
			continue
		}
		data = append(data, item)
	}

	bytes, err := tmjson.Marshal(data)
	if err != nil {
		return err
	}

	store.Lock()
	defer store.Unlock()
	if err := store.db.Set(uint32ToBytes(store.pending.height), bytes); err != nil {
		return err
	}
	return nil
}

func (store *eventsStore) loadCache() {
	store.Lock()
	if len(store.idPubKey) == 0 {
		store.loadPubKeys()
		store.loadAddresses()
	}
	store.Unlock()
}

const pubKeyPrefix = "pubKey"
const addressPrefix = "address"
const pubKeysCountKey = "pubKeys"
const addressesCountKey = "addresses"

func (store *eventsStore) saveAddress(address [20]byte) uint32 {

	if id, ok := store.addressID[address]; ok {
		return id
	}

	id := uint32(len(store.addressID))
	store.cacheAddress(id, address)

	if err := store.db.Set(append([]byte(addressPrefix), uint32ToBytes(id)...), address[:]); err != nil {
		panic(err)
	}
	if err := store.db.Set([]byte(addressesCountKey), uint32ToBytes(uint32(len(store.addressID)))); err != nil {
		panic(err)
	}
	return id
}

func (store *eventsStore) savePubKey(validatorPubKey [32]byte) uint16 {

	key := validatorPubKey
	if id, ok := store.pubKeyID[key]; ok {
		return id
	}

	id := uint16(len(store.idPubKey))
	store.cachePubKey(id, key)

	if err := store.db.Set(append([]byte(pubKeyPrefix), uint16ToBytes(id)...), validatorPubKey[:]); err != nil {
		panic(err)
	}
	if err := store.db.Set([]byte(pubKeysCountKey), uint16ToBytes(uint16(len(store.idPubKey)))); err != nil {
		panic(err)
	}
	return id
}

func (store *eventsStore) loadPubKeys() {
	if count, _ := store.db.Get([]byte(pubKeysCountKey)); len(count) > 0 {
		for id := uint16(0); id < binary.BigEndian.Uint16(count); id++ {
			key, err := store.db.Get(append([]byte(pubKeyPrefix), uint16ToBytes(id)...))
			if err != nil {
				panic(err)
			}
			var pubKey [32]byte
			copy(pubKey[:], key)
			store.cachePubKey(id, pubKey)
		}
	}
}

func (store *eventsStore) loadAddresses() {
	count, err := store.db.Get([]byte(addressesCountKey))
	if err != nil {
		panic(err)
	}
	if len(count) > 0 {
		for id := uint32(0); id < binary.BigEndian.Uint32(count); id++ {
			address, _ := store.db.Get(append([]byte(addressPrefix), uint32ToBytes(id)...))
			var key [20]byte
			copy(key[:], address)
			store.cacheAddress(id, key)
		}
	}
}

func uint32ToBytes(height uint32) []byte {
	var h = make([]byte, 4)
	binary.BigEndian.PutUint32(h, height)
	return h
}

func uint16ToBytes(height uint16) []byte {
	var h = make([]byte, 2)
	binary.BigEndian.PutUint16(h, height)
	return h
}
