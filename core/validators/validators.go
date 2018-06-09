package validators

func GetValidatorsCountForBlock(block uint64) int {
	count := 16 + (block/518400)*4

	if count > 256 {
		return 256
	}

	return int(count)
}
