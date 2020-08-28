package upgrades

var gracePeriods = []*gracePeriod{
	NewGracePeriod(1, 120),
	NewGracePeriod(UpgradeBlock1, UpgradeBlock1+120),
	NewGracePeriod(UpgradeBlock2, UpgradeBlock2+120),
	NewGracePeriod(UpgradeBlock3, UpgradeBlock3+120),
	NewGracePeriod(UpgradeBlock4, UpgradeBlock4+120),
}

type gracePeriod struct {
	from uint64
	to   uint64
}

func (gp *gracePeriod) IsApplicable(block uint64) bool {
	return block >= gp.from && block <= gp.to
}

func NewGracePeriod(from uint64, to uint64) *gracePeriod {
	return &gracePeriod{from: from, to: to}
}

func IsGraceBlock(block uint64) bool {
	for _, gp := range gracePeriods {
		if gp.IsApplicable(block) {
			return true
		}
	}

	return false
}
