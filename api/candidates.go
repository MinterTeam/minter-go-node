package api

func Candidates(height int, includeStakes bool) (*[]CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	if height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		if includeStakes {
			cState.Candidates.LoadStakes()
		}
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidates := cState.Candidates.GetCandidates()

	result := make([]CandidateResponse, len(candidates))
	for i, candidate := range candidates {
		result[i] = makeResponseCandidate(cState, *candidate, includeStakes)
	}

	return &result, nil
}
