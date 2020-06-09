package events

import (
	"encoding/binary"
	"github.com/tendermint/go-amino"
	db "github.com/tendermint/tm-db"
	"sync"
)

type IEventsDB interface {
	AddEvent(height uint32, event Event)
	LoadEvents(height uint32) Events
	CommitEvents() error
}

type eventsStore struct {
	cdc *amino.Codec
	sync.RWMutex
	db        db.DB
	pending   pendingEvents
	idPubKey  map[uint16]string
	pubKeyID  map[string]uint16
	idAddress map[uint32][20]byte
	addressID map[[20]byte]uint32
}

type pendingEvents struct {
	sync.Mutex
	height uint32
	items  Events
}

func NewEventsStore(db db.DB) IEventsDB {
	codec := amino.NewCodec()
	codec.RegisterInterface((*interface{})(nil), nil)
	codec.RegisterConcrete(reward{}, "reward", nil)
	codec.RegisterConcrete(slash{}, "slash", nil)
	codec.RegisterConcrete(unbond{}, "unbond", nil)

	return &eventsStore{
		cdc:       codec,
		RWMutex:   sync.RWMutex{},
		db:        db,
		pending:   pendingEvents{},
		idPubKey:  make(map[uint16]string),
		pubKeyID:  make(map[string]uint16),
		idAddress: make(map[uint32][20]byte),
		addressID: make(map[[20]byte]uint32),
	}
}

func (store *eventsStore) cachePubKey(id uint16, key string) {
	store.idPubKey[id] = key
	store.pubKeyID[key] = id
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

	bytes, err := store.db.Get(uint32ToBytes(height))
	if err != nil {
		panic(err)
	}
	if len(bytes) == 0 {
		return Events{}
	}

	var items []interface{}
	if err := store.cdc.UnmarshalBinaryBare(bytes, &items); err != nil {
		panic(err)
	}

	resultEvents := make(Events, 0, len(items))
	for _, event := range items {
		resultEvents = append(resultEvents, store.compile(event))
	}

	return resultEvents
}

func (store *eventsStore) CommitEvents() error {
	store.loadCache()

	store.pending.Lock()
	defer store.pending.Unlock()
	var data []interface{}
	for _, item := range store.pending.items {
		data = append(data, store.convert(item))
	}

	bytes, err := store.cdc.MarshalBinaryBare(data)
	if err != nil {
		return err
	}

	store.Lock()
	defer store.Unlock()
	if err := store.db.Set(uint32ToBytes(store.pending.height), bytes); err != nil {
		panic(err)
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

func (store *eventsStore) convert(event Event) interface{} {
	var res interface{}
	switch event.(type) {
	case RewardEvent:
		res = store.convertReward(event.(RewardEvent))
	case UnbondEvent:
		res = store.convertUnbound(event.(UnbondEvent))
	case SlashEvent:
		res = store.convertSlash(event.(SlashEvent))
	default:
		res = event
	}
	return res
}

func (store *eventsStore) convertReward(rewardEvent RewardEvent) interface{} {
	return rewardConvert(&rewardEvent, store.savePubKey(rewardEvent.ValidatorPubKey), store.saveAddress(rewardEvent.Address))
}

func (store *eventsStore) convertUnbound(unbondEvent UnbondEvent) interface{} {
	return convertUnbound(&unbondEvent, store.savePubKey(unbondEvent.ValidatorPubKey), store.saveAddress(unbondEvent.Address))
}

func (store *eventsStore) convertSlash(slashEvent SlashEvent) interface{} {
	return convertSlash(&slashEvent, store.savePubKey(slashEvent.ValidatorPubKey), store.saveAddress(slashEvent.Address))
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

	key := string(validatorPubKey[:])
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
			pubKey, err := store.db.Get(append([]byte(pubKeyPrefix), uint16ToBytes(id)...))
			if err != nil {
				panic(err)
			}
			store.cachePubKey(id, string(pubKey))
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

func (store *eventsStore) compile(event interface{}) Event {
	var res interface{}
	switch event.(type) {
	case reward:
		res = store.compileReward(event.(reward))
	case unbond:
		res = store.compileUnbond(event.(unbond))
	case slash:
		res = store.compileSlash(event.(slash))
	default:
		res = event
	}
	return res
}

func (store *eventsStore) compileReward(item reward) interface{} {
	return compileReward(&item, store.idPubKey[item.PubKeyID], store.idAddress[item.AddressID])
}

func (store *eventsStore) compileSlash(item slash) interface{} {
	return compileSlash(&item, store.idPubKey[item.PubKeyID], store.idAddress[item.AddressID])
}

func (store *eventsStore) compileUnbond(item unbond) interface{} {
	return compileUnbond(&item, store.idPubKey[item.PubKeyID], store.idAddress[item.AddressID])
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
