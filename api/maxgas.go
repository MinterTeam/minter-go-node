package api

func MaxGas(height int) (*uint64, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	maxGas := cState.GetCurrentMaxGas()
	return &maxGas, nil
}
