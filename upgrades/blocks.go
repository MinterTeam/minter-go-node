package upgrades

const UpgradeBlock1 = 1184805

func IsUpgradeBlock(height uint64) bool {
	upgradeBlocks := []uint64{UpgradeBlock1}

	for _, block := range upgradeBlocks {
		if height == block {
			return true
		}
	}

	return false
}
