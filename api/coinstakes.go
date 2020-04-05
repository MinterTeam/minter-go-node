package api

func Coinstakes(height int, symbol string) (*[]CoinstakeResponse, error) {
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


	ggg := 0
	tmp := "1"

	hhh := make([]CoinstakeResponse, len(candidates))





	for i, candidate := range candidates {
			if len(makeResponseCoinstake(cState, *candidate, symbol)) != len(hhh[i])  {
			tmp = tmp + "1" 
		}
	}




	
	result := make([]CoinstakeResponse, len(tmp)-1)



	for i, candidate := range candidates {
		if len(makeResponseCoinstake(cState, *candidate, symbol)) != len(hhh[i]) {
		result[ggg] = makeResponseCoinstake(cState, *candidate, symbol)
i=i
		ggg = ggg + 1
}

	}

	return &result, nil
}
