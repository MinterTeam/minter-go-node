package api

func Candidates(height int, includeStakes bool) (*[]CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	candidates := cState.GetStateCandidates().GetData()

	result := make([]CandidateResponse, len(candidates))
	for i, candidate := range candidates {
		result[i] = makeResponseCandidate(candidate, includeStakes)
	}

	return &result, nil
}
