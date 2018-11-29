package api

func Candidates(height int) (*[]CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	candidates := cState.GetStateCandidates().GetData()

	var result []CandidateResponse
	for _, candidate := range candidates {
		result = append(result, makeResponseCandidate(candidate, false))
	}

	return &result, nil
}
