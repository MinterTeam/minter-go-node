package api

import (
	"github.com/pkg/errors"
)

func Candidate(pubkey []byte, height int) (*CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	candidate := cState.GetStateCandidate(pubkey)
	if candidate == nil {
		return nil, errors.New("Candidate not found")
	}

	response := makeResponseCandidate(*candidate, true)
	return &response, nil
}
