package api

func mCandidates(height int) (*[]mCandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	if height != 0 {
		cState.Lock()
		cState.Candidates.LoadCandidates()
		cState.Candidates.LoadStakes()
		cState.Unlock()
	}

	cState.RLock()
	defer cState.RUnlock()

	candidates := cState.Candidates.GetCandidates()
	
	result := make([]mCandidateResponse, len(candidates))
	for i, candidate := range candidates {

		result[i] = mmakeResponseCandidate(cState, *candidate)

 	}
	return &result, nil
}
