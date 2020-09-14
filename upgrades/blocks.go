package upgrades

const UpgradeBlock1 = 0
const UpgradeBlock2 = 0
const UpgradeBlock3 = 0
const UpgradeBlock4 = 0

func IsUpgradeBlock(height uint64) bool {
	upgradeBlocks := []uint64{UpgradeBlock1, UpgradeBlock2, UpgradeBlock3, UpgradeBlock4}

	for _, block := range upgradeBlocks {
		if height == block {
			return true
		}
	}

	return false
}
