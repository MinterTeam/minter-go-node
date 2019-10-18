package api

func Candidates(height int, includeStakes bool) (*[]CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	candidates := cState.Candidates.GetCandidates()

	result := make([]CandidateResponse, len(candidates))
	for i, candidate := range candidates {
		result[i] = makeResponseCandidate(cState, *candidate, includeStakes)
	}

	return &result, nil
}
