package checks

import (
	"github.com/MinterTeam/minter-go-node/core/check"
	db "github.com/tendermint/tm-db"
)

type Checks struct {
	db db.DB
}

func NewChecks(db db.DB) (*Checks, error) {
	return &Checks{db: db}, nil
}

func (v *Checks) Commit() error {
	panic("implement me")
}

func (v *Checks) IsCheckUsed(check *check.Check) bool {
	panic("implement me")
}

func (v *Checks) UseCheck(check *check.Check) {
	panic("implement me")
}
