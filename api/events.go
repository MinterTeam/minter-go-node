package api

import (
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/eventsdb/events"
	"github.com/tendermint/tendermint/libs/db"
)

type EventsResponse struct {
	Events events.Events `json:"events"`
}

func Events(height uint64) (*EventsResponse, error) {
	return &EventsResponse{
		Events: eventsdb.NewEventsDB(db.NewDB("events", db.DBBackendType(minterCfg.DBBackend), utils.GetMinterHome()+"/data")).LoadEvents(height),
	}, nil
}
