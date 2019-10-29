package checks

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/check"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/tree"
)

const mainPrefix = byte('t')

type Checks struct {
	usedChecks map[types.Hash]struct{}

	iavl tree.Tree
}

func NewChecks(iavl tree.Tree) (*Checks, error) {
	return &Checks{iavl: iavl, usedChecks: map[types.Hash]struct{}{}}, nil
}

func (c *Checks) Commit() error {
	for hash := range c.usedChecks {
		delete(c.usedChecks, hash)

		trieHash := append([]byte{mainPrefix}, hash.Bytes()...)
		c.iavl.Set(trieHash, []byte{0x1})
	}

	return nil
}

func (c *Checks) IsCheckUsed(check *check.Check) bool {
	if _, has := c.usedChecks[check.Hash()]; has {
		return true
	}

	_, data := c.iavl.Get(append([]byte{mainPrefix}, check.Hash().Bytes()...))

	return len(data) != 0
}

func (c *Checks) UseCheck(check *check.Check) {
	c.UseCheckHash(check.Hash())

}

func (c *Checks) UseCheckHash(hash types.Hash) {
	c.usedChecks[hash] = struct{}{}
}

func (c *Checks) Export(state *types.AppState) {
	// todo: iterate range?
	c.iavl.Iterate(func(key []byte, value []byte) bool {
		if key[0] == mainPrefix {
			state.UsedChecks = append(state.UsedChecks, types.UsedCheck(fmt.Sprintf("%x", key[1:])))
		}

		return false
	})
}
