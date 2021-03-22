package upgrades

type Grace struct {
	gracePeriods []*GracePeriod
}

func (g *Grace) AddGracePeriods(gracePeriods ...*GracePeriod) {
	g.gracePeriods = append(g.gracePeriods, gracePeriods...)
}

func NewGrace() *Grace {
	return &Grace{}
}

func (g *Grace) IsGraceBlock(block uint64) bool {
	if g == nil {
		return false
	}
	for _, gp := range g.gracePeriods {
		if gp.isApplicable(block) {
			return true
		}
	}

	return false
}

type GracePeriod struct {
	from uint64
	to   uint64
}

func (gp *GracePeriod) isApplicable(block uint64) bool {
	return block >= gp.from && block <= gp.to
}

func NewGracePeriod(from uint64, to uint64) *GracePeriod {
	return &GracePeriod{from: from, to: to}
}
