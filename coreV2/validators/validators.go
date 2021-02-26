package validators

// GetValidatorsCountForBlock returns available validators slots for given height
func GetValidatorsCountForBlock(block uint64) int {
	return 64
}

// GetCandidatesCountForBlock returns available candidates slots for given height
func GetCandidatesCountForBlock(block uint64) int {
	return 192
}
