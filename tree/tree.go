package tree

import (
	"fmt"
	"sync"

	"github.com/cosmos/iavl"
	dbm "github.com/tendermint/tm-db"
)

type saver interface {
	Commit(db *iavl.MutableTree, version int64) error
	SetImmutableTree(immutableTree *iavl.ImmutableTree)
	// ModuleName() string // todo
}

// MTree mutable tree, used for txs delivery
type MTree interface {
	Commit(...saver) ([]byte, int64, error)
	GetLastImmutable() *iavl.ImmutableTree
	GetImmutableAtHeight(version int64) (*iavl.ImmutableTree, error)

	Export(version int64) (*iavl.Exporter, error)
	Import(version int64) (*iavl.Importer, error)

	DeleteVersion(version int64) error
	DeleteVersionsRange(fromVersion, toVersion int64) error

	AvailableVersions() []int
	Version() int64
}

func (t *mutableTree) Commit(savers ...saver) (hash []byte, version int64, err error) {
	v := t.Version()
	t.lock.Lock()
	defer t.lock.Unlock()

	for _, saver := range savers {
		err := saver.Commit(t.tree, v)
		if err != nil {
			return nil, 0, err
			// return nil, 0, errors.Wrap(err, saver.ModuleName())
		}
	}

	hash, version, err = t.tree.SaveVersion()
	if err != nil {
		return nil, 0, err
	}

	immutable, err := t.tree.GetImmutable(t.tree.Version())
	if err != nil {
		return nil, 0, err
	}
	for _, saver := range savers {
		saver.SetImmutableTree(immutable)
	}

	return hash, version, err
}

// Import imports an IAVL tree at the given version, returning an iavl.Importer for importing.
func (t *mutableTree) Import(version int64) (*iavl.Importer, error) {
	return t.tree.Import(version)
}

// Exports the IAVL store at the given version, returning an iavl.Exporter for the tree.
func (t *mutableTree) Export(version int64) (*iavl.Exporter, error) {
	itree, err := t.GetImmutableAtHeight(version)
	if err != nil {
		return nil, fmt.Errorf("iavl export failed for version %v: %w", version, err)
	}
	if itree == nil {
		return nil, fmt.Errorf("iavl export failed: unable to fetch tree for version %v", version)
	}
	return itree.Export(), nil
}

func (t *mutableTree) MutableTree() *iavl.MutableTree {
	return t.tree
}

// NewMutableTree creates and returns new MutableTree using given db. Panics on error.
// If you want to get read-only state, you should use height = 0 and LazyLoadVersion (version), see NewImmutableTree
func NewMutableTree(height uint64, db dbm.DB, cacheSize int, initialVersion uint64) (MTree, error) {
	tree, err := iavl.NewMutableTreeWithOpts(db, cacheSize, &iavl.Options{InitialVersion: initialVersion})
	if err != nil {
		return nil, err
	}
	m := &mutableTree{
		tree: tree,
	}
	if height < initialVersion {
		return m, nil
	}

	if _, err := m.tree.LoadVersion(int64(height)); err != nil {
		return nil, err
	}

	return m, nil
}

type mutableTree struct {
	tree *iavl.MutableTree
	lock sync.RWMutex
}

func (t *mutableTree) GetImmutableAtHeight(version int64) (*iavl.ImmutableTree, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	tree, err := t.tree.GetImmutable(version)
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func (t *mutableTree) Version() int64 {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Version()
}

func (t *mutableTree) GetLastImmutable() *iavl.ImmutableTree {
	t.lock.RLock()
	defer t.lock.RUnlock()

	immutable, err := t.tree.GetImmutable(t.tree.Version())
	if err != nil {
		return iavl.NewImmutableTree(dbm.NewMemDB(), 0)
	}

	return immutable
}

// Should use GlobalLock() and GlobalUnlock
func (t *mutableTree) DeleteVersionsRange(fromVersion, toVersion int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	err := t.tree.DeleteVersionsRange(fromVersion, toVersion)
	if err != nil {
		return err
	}

	return nil
}

func (t *mutableTree) DeleteVersion(version int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.existVersion(version) {
		return t.tree.DeleteVersion(version)
	}
	return nil
}

func (t *mutableTree) existVersion(version int64) bool {
	return t.tree.VersionExists(version)
}

func (t *mutableTree) AvailableVersions() []int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.AvailableVersions()
}

// NewImmutableTree returns iavl.ImmutableTree from given db at given height
func NewImmutableTree(height uint64, db dbm.DB) (*iavl.ImmutableTree, error) {
	tree, err := iavl.NewMutableTree(db, 1024)
	if err != nil {
		return nil, err
	}
	immutableTree, err := tree.GetImmutable(int64(height))
	if err != nil {
		return nil, err
	}
	return immutableTree, nil
}
