package api

import (
	compact "github.com/klim0v/compact-db"
)

type EventsResponse struct {
	Events compact.Events `json:"events"`
}

func Events(height uint64) (*EventsResponse, error) {
	return &EventsResponse{
		Events: blockchain.GetEventsDB().LoadEvents(uint32(height)),
	}, nil
}
