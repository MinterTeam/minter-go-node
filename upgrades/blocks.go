package upgrades

func IsUpgradeBlock(height uint64) bool {
	upgradeBlocks := []uint64{} // fill this

	for _, block := range upgradeBlocks {
		if height == block {
			return true
		}
	}

	return false
}
