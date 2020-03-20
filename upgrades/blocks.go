package upgrades

const UpgradeBlock1 = 5000
const UpgradeBlock2 = 38519
const UpgradeBlock3 = 109000

func IsUpgradeBlock(height uint64) bool {
	upgradeBlocks := []uint64{UpgradeBlock1, UpgradeBlock2, UpgradeBlock3}

	for _, block := range upgradeBlocks {
		if height == block {
			return true
		}
	}

	return false
}
