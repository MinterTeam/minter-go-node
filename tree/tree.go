package tree

import (
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"sync"
)

// ReadOnlyTree used for CheckState: API and CheckTx calls. Immutable.
// Deprecated: Use GetLastImmutable().Get(key)
type ReadOnlyTree interface {
	// Deprecated: Use GetLastImmutable()
	Get(key []byte) (index int64, value []byte)
	// Deprecated: Use GetLastImmutable()
	Version() int64
	// Deprecated: Use GetLastImmutable()
	Hash() []byte
	// Deprecated: Use GetLastImmutable()
	Iterate(fn func(key []byte, value []byte) bool) (stopped bool)
	// Deprecated: Use GetLastImmutable()
	AvailableVersions() []int
}

type Saver interface {
	Commit(db *iavl.MutableTree) error
	SetImmutableTree(immutableTree *iavl.ImmutableTree)
}

// MTree mutable tree, used for txs delivery
type MTree interface {
	Commit(...Saver) ([]byte, int64, error)
	MutableTree() *iavl.MutableTree
	GetLastImmutable() *iavl.ImmutableTree
	GetImmutableAtHeight(version int64) (*iavl.ImmutableTree, error)

	DeleteVersion(version int64) error
	DeleteVersionsRange(fromVersion, toVersion int64) error

	ReadOnlyTree
	Set(key, value []byte) bool
	Remove(key []byte) ([]byte, bool)
	LoadVersion(targetVersion int64) (int64, error)
	LazyLoadVersion(targetVersion int64) (int64, error)
	SaveVersion() ([]byte, int64, error)
}

func (t *mutableTree) Commit(savers ...Saver) (hash []byte, version int64, err error) {
	t.Lock()
	defer t.Unlock()

	for _, saver := range savers {
		err := saver.Commit(t.MutableTree())
		if err != nil {
			return nil, 0, err
		}
	}

	hash, version, err = t.SaveVersion()

	immutable := t.GetLastImmutable()
	for _, saver := range savers {
		saver.SetImmutableTree(immutable)
	}

	return hash, version, err
}
func (t *mutableTree) MutableTree() *iavl.MutableTree {
	return t.tree
}

// NewMutableTree creates and returns new MutableTree using given db. Panics on error.
// If you want to get read-only state, you should use height = 0 and LazyLoadVersion (version), see NewImmutableTree
func NewMutableTree(height uint64, db dbm.DB, cacheSize int) (MTree, error) {
	tree, err := iavl.NewMutableTree(db, cacheSize)
	if err != nil {
		return nil, err
	}

	m := &mutableTree{
		tree: tree,
	}
	if height == 0 {
		return m, nil
	}

	if _, err := m.tree.LoadVersionForOverwriting(int64(height)); err != nil {
		return nil, err
	}

	return m, nil
}

type mutableTree struct {
	tree *iavl.MutableTree
	lock sync.RWMutex
	sync.Mutex
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

func (t *mutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Iterate(fn)
}

func (t *mutableTree) Hash() []byte {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Hash()
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

// Deprecated: Use GetLastImmutable().Get(key)
func (t *mutableTree) Get(key []byte) (index int64, value []byte) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Get(key)
}

func (t *mutableTree) Set(key, value []byte) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.Set(key, value)
}

func (t *mutableTree) Remove(key []byte) ([]byte, bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.Remove(key)
}

func (t *mutableTree) LoadVersion(targetVersion int64) (int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.LoadVersion(targetVersion)
}

func (t *mutableTree) LazyLoadVersion(targetVersion int64) (int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.LazyLoadVersion(targetVersion)
}

// Should use GlobalLock() and GlobalUnlock
func (t *mutableTree) SaveVersion() ([]byte, int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.SaveVersion()
}

// Should use GlobalLock() and GlobalUnlock
func (t *mutableTree) DeleteVersionsRange(fromVersion, toVersion int64) error {
	t.Lock()
	defer t.Unlock()

	err := t.tree.DeleteVersionsRange(fromVersion, toVersion)
	if err != nil {
		return err
	}

	return nil
}

func (t *mutableTree) DeleteVersion(version int64) error {
	t.Lock()
	defer t.Unlock()

	return t.tree.DeleteVersion(version)
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
	// _, err = tree.LazyLoadVersion(int64(height))
	// if err != nil {
	// 	return nil, err
	// }
	immutableTree, err := tree.GetImmutable(int64(height))
	if err != nil {
		return nil, err
	}
	return immutableTree, nil
}
