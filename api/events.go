package api

import (
	"github.com/MinterTeam/minter-go-node/eventsdb"
)

var edb *eventsdb.EventsDB

func init() {
	edb = eventsdb.NewEventsDB(eventsdb.GetCurrentDB())
}

type EventsResponse struct {
	Events eventsdb.Events `json:"events"`
}

func Events(height int64) (*EventsResponse, error) {
	return &EventsResponse{
		Events: edb.LoadEvents(height),
	}, nil
}
