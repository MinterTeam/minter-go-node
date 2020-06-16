package api

import (
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
)

type FindEventsResponse struct {
	Events eventsdb.Events `json:"events"`
}

func FindEvents(search []string, height uint64) (*FindEventsResponse, error) {
	var result FindEventsResponse
	events := blockchain.GetEventsDB().LoadEvents(uint32(height))

	for _, event := range events {
		for _, addr := range search {
			if event.AddressString() == addr || event.ValidatorPubKeyString() == addr {
				result.Events = append(result.Events, event)
			}
		}
	}

	return &result, nil
}
