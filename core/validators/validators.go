package validators

func GetValidatorsCountForBlock(block uint64) int {
	return 64
}

func GetCandidatesCountForBlock(block uint64) int {
	return GetValidatorsCountForBlock(block) * 3
}
