package validators

var startHeight uint64 = 0

func GetValidatorsCountForBlock(block uint64) int {
	block += startHeight
	count := 16 + (block/518400)*4

	if count > 256 {
		return 256
	}

	return int(count)
}

func GetCandidatesCountForBlock(block uint64) int {
	return GetValidatorsCountForBlock(block) * 3
}

func SetStartHeight(sHeight uint64) {
	startHeight = sHeight
}
