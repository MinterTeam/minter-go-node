package eventsdb

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/mintdb"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tendermint/go-amino"
)

var cdc = amino.NewCodec()

var edb *EventsDB

func init() {
	RegisterAminoEvents(cdc)

	eventsDB, err := mintdb.NewLDBDatabase(utils.GetMinterHome()+"/events", 1000, 1000)

	if err != nil {
		panic(err)
	}

	edb = NewEventsDB(eventsDB)
}

func GetCurrent() *EventsDB {
	return edb
}

type EventsDB struct {
	db *mintdb.LDBDatabase
}

func NewEventsDB(db *mintdb.LDBDatabase) *EventsDB {
	return &EventsDB{
		db: db,
	}
}

func (db *EventsDB) SaveEvent(height int64, event Event) error {
	events := db.GetEvents(height)
	events = append(events, event)

	return db.SaveEvents(height, events)
}

func (db *EventsDB) SaveEvents(height int64, events Events) error {
	key := getKeyForHeight(height)

	bytes, err := cdc.MarshalBinary(events)

	if err != nil {
		return err
	}

	return db.db.Put(key, bytes)
}

func (db *EventsDB) GetEvents(height int64) Events {
	key := getKeyForHeight(height)

	data, err := db.db.Get(key)

	if err != nil {
		if err == leveldb.ErrNotFound {
			return Events{}
		}

		panic(err)
	}

	var decoded Events
	err = cdc.UnmarshalBinary(data, &decoded)

	if err != nil {
		panic(err)
	}

	return decoded
}

func getKeyForHeight(height int64) []byte {
	var h = make([]byte, 8)
	binary.BigEndian.PutUint64(h, uint64(height))

	return h
}
