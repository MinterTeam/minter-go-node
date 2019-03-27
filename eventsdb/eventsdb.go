package eventsdb

import (
	"encoding/binary"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/tendermint/libs/db"
	"sync"
)

var cdc = amino.NewCodec()

var eventsEnabled = !config.GetConfig().ValidatorMode

var edb *EventsDB
var godb *db.GoLevelDB

func init() {
	RegisterAminoEvents(cdc)
}

func GetCurrentDB() *db.GoLevelDB {
	if godb != nil {
		return godb
	}

	gdb, err := db.NewGoLevelDB("events", utils.GetMinterHome()+"/data")

	if err != nil {
		panic(err)
	}

	godb = gdb

	return gdb
}

func GetCurrent() *EventsDB {
	if edb != nil {
		return edb
	}

	edb = NewEventsDB(GetCurrentDB())

	return edb
}

type EventsDB struct {
	db    *db.GoLevelDB
	cache *eventsCache

	lock sync.RWMutex
}

type eventsCache struct {
	height uint64
	events Events

	lock sync.RWMutex
}

func (c *eventsCache) set(height uint64, events Events) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.height, c.events = height, events
}

func (c *eventsCache) get() Events {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.events
}

func (c *eventsCache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.height = 0
	c.events = nil
}

func NewEventsDB(db *db.GoLevelDB) *EventsDB {
	return &EventsDB{
		db: db,
		cache: &eventsCache{
			height: 0,
			events: nil,
			lock:   sync.RWMutex{},
		},
		lock: sync.RWMutex{},
	}
}

func (db *EventsDB) AddEvent(height uint64, event Event) {
	if !eventsEnabled {
		return
	}

	events := db.getEvents(height)
	db.setEvents(height, append(events, event))
}

func (db *EventsDB) FlushEvents() error {
	if !eventsEnabled {
		return nil
	}

	height := db.cache.height

	events := db.getEvents(height)
	bytes, err := cdc.MarshalBinaryBare(events)

	if err != nil {
		return err
	}

	db.cache.Clear()
	db.db.Set(getKeyForHeight(height), bytes)

	return nil
}

func (db *EventsDB) setEvents(height uint64, events Events) {
	db.cache.set(height, events)
}

func (db *EventsDB) LoadEvents(height uint64) Events {
	db.lock.RLock()
	data := db.db.Get(getKeyForHeight(height))
	db.lock.RUnlock()

	if len(data) == 0 {
		return Events{}
	}

	var decoded Events
	err := cdc.UnmarshalBinaryBare(data, &decoded)

	if err != nil {
		panic(err)
	}

	return decoded
}

func (db *EventsDB) getEvents(height uint64) Events {
	if db.cache.height == height {
		return db.cache.get()
	}

	events := db.LoadEvents(height)
	db.cache.set(height, events)

	return events
}

func getKeyForHeight(height uint64) []byte {
	var h = make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)

	return h
}
