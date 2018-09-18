package eventsdb

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/db"
)

var cdc = amino.NewCodec()

var eventsEnabled = config.GetConfig().EnableEvents

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
	cache map[int64]Events
}

func NewEventsDB(db *db.GoLevelDB) *EventsDB {
	return &EventsDB{
		db:    db,
		cache: map[int64]Events{},
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

	delete(db.cache, height)

	db.db.Set(key, bytes)

	return nil
}

func (db *EventsDB) SetEvents(height int64, events Events) {
	db.cache[height] = events
}

func (db *EventsDB) GetEvents(height int64) Events {

	if events, has := db.cache[height]; has {
		return events
	}

	key := getKeyForHeight(height)

	data := db.db.Get(key)

	var decoded Events
	err := cdc.UnmarshalBinary(data, &decoded)

	if err != nil {
		panic(err)
	}

	db.cache[height] = decoded

	return decoded
}

func getKeyForHeight(height int64) []byte {
	var h = make([]byte, 8)
	binary.BigEndian.PutUint64(h, uint64(height))

	return h
}
