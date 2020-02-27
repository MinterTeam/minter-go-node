package upgrades

var gracePeriods = []*gracePeriod{
	NewGracePeriod(1, 15),
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
