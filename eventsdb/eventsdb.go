package eventsdb

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/db"
	"sync"
)

var cdc = amino.NewCodec()

var eventsEnabled = !config.GetConfig().ValidatorMode

var edb *EventsDB

func init() {
	RegisterAminoEvents(cdc)
}

func GetCurrent() *EventsDB {
	if edb == nil {
		eventsDB, err := db.NewGoLevelDB("events", utils.GetMinterHome()+"/data")

		if err != nil {
			panic(err)
		}

		edb = NewEventsDB(eventsDB)
	}

	return edb
}

type EventsDB struct {
	db    *db.GoLevelDB
	cache *EventsCache

	lock sync.RWMutex
}

type EventsCache struct {
	height int64
	events Events

	lock sync.RWMutex
}

func (c *EventsCache) Set(height int64, events Events) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.height, c.events = height, events
}

func (c *EventsCache) Get() Events {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.events
}

func (c *EventsCache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.height = 0
	c.events = nil
}

func NewEventsDB(db *db.GoLevelDB) *EventsDB {
	return &EventsDB{
		db: db,
		cache: &EventsCache{
			height: 0,
			events: nil,
			lock:   sync.RWMutex{},
		},
		lock: sync.RWMutex{},
	}
}

func (db *EventsDB) AddEvent(height int64, event Event) {
	if !eventsEnabled {
		return
	}

	events := db.GetEvents(height)
	db.SetEvents(height, append(events, event))
}

func (db *EventsDB) FlushEvents(height int64) error {
	if !eventsEnabled {
		return nil
	}

	events := db.GetEvents(height)

	key := getKeyForHeight(height)

	bytes, err := cdc.MarshalBinary(events)

	if err != nil {
		return err
	}

	db.cache.Clear()

	db.lock.Lock()
	db.db.Set(getKeyForHeight(height), bytes)
	db.lock.Unlock()

	return nil
}

func (db *EventsDB) SetEvents(height int64, events Events) {
	db.cache.Set(height, events)
}

func (db *EventsDB) LoadEvents(height int64) Events {
	db.lock.RLock()
	defer db.lock.RUnlock()

	data := db.db.Get(getKeyForHeight(height))

	if len(data) == 0 {
		return Events{}
	}

	var decoded Events
	err := cdc.UnmarshalBinary(data, &decoded)

	if err != nil {
		panic(err)
	}

	return decoded
}

func (db *EventsDB) GetEvents(height int64) Events {
	if db.cache.height == height {
		return db.cache.Get()
	}

	events := db.LoadEvents(height)

	db.cache.Set(height, events)

	return events
}

func getKeyForHeight(height int64) []byte {
	var h = make([]byte, 8)
	binary.BigEndian.PutUint64(h, uint64(height))

	return h
}
