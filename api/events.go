package api

import (
	compact "github.com/klim0v/compact-db"
)

type EventsResponse struct {
	Events compact.Events `json:"events"`
}

func Events(height uint32) (*EventsResponse, error) {
	return &EventsResponse{
		Events: blockchain.GetEventsDB().LoadEvents(height),
	}, nil
}
