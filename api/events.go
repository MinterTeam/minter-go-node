package api

import (
	eventsdb "github.com/MinterTeam/events-db"
)

type EventsResponse struct {
	Events eventsdb.Events `json:"events"`
}

func Events(height uint64) (*EventsResponse, error) {
	return &EventsResponse{
		Events: blockchain.GetEventsDB().LoadEvents(uint32(height)),
	}, nil
}
