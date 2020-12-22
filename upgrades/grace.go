package upgrades

var gracePeriods = []*gracePeriod{
	newGracePeriod(1, 120),
	newGracePeriod(UpgradeBlock1, UpgradeBlock1+120),
}

func IsGraceBlock(block uint64) bool {
	for _, gp := range gracePeriods {
		if gp.isApplicable(block) {
			return true
		}
	}

	return false
}

type gracePeriod struct {
	from uint64
	to   uint64
}

func (gp *gracePeriod) isApplicable(block uint64) bool {
	return block >= gp.from && block <= gp.to
}

func newGracePeriod(from uint64, to uint64) *gracePeriod {
	return &gracePeriod{from: from, to: to}
}
