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
}

func (c *EventsCache) Set(height int64, events Events) {
	c.height = height
	c.events = events
}

func NewEventsDB(db *db.GoLevelDB) *EventsDB {
	return &EventsDB{
		db:    db,
		cache: &EventsCache{},
	}
}

func (db *EventsDB) AddEvent(height int64, event Event) {
	if !eventsEnabled {
		return
	}

	events := db.GetEvents(height)
	events = append(events, event)

	db.SetEvents(height, events)
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

	db.cache = &EventsCache{}

	db.db.Set(key, bytes)

	return nil
}

func (db *EventsDB) SetEvents(height int64, events Events) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.cache.Set(height, events)
}

func (db *EventsDB) LoadEvents(height int64) Events {
	key := getKeyForHeight(height)

	data := db.db.Get(key)

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
		return db.cache.events
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
