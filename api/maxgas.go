package api

func MaxGas(height int) (*uint64, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.Lock()
	defer cState.Unlock()

	maxGas := cState.App.GetMaxGas()
	return &maxGas, nil
}
