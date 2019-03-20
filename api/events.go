package api

import (
	"github.com/MinterTeam/minter-go-node/eventsdb"
)

type EventsResponse struct {
	Events eventsdb.Events `json:"events"`
}

func Events(height uint64) (*EventsResponse, error) {
	return &EventsResponse{
		Events: eventsdb.NewEventsDB(eventsdb.GetCurrentDB()).LoadEvents(height),
	}, nil
}
