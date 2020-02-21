package api

func MaxGas(height int) (*uint64, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	cState.RLock()
	defer cState.RUnlock()

	maxGas := cState.App.GetMaxGas()
	return &maxGas, nil
}
