package tree

import (
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"strings"
	"sync"
)

// ReadOnlyTree used for CheckState: API and CheckTx calls. Immutable.
type ReadOnlyTree interface {
	Get(key []byte) (index int64, value []byte)
	Version() int64
	Hash() []byte
	Iterate(fn func(key []byte, value []byte) bool) (stopped bool)
	AvailableVersions() []int
	KeepLastHeight() int64
}

// MTree mutable tree, used for txs delivery
type MTree interface {
	ReadOnlyTree
	Set(key, value []byte) bool
	Remove(key []byte) ([]byte, bool)
	LoadVersion(targetVersion int64) (int64, error)
	LazyLoadVersion(targetVersion int64) (int64, error)
	SaveVersion() ([]byte, int64, error)
	DeleteVersionIfExists(version int64) error
	DeleteVersionsFromInterval(fromVersion, toVersion int64) error
	GetImmutable() *ImmutableTree
	GetImmutableAtHeight(version int64) (*ImmutableTree, error)
	GlobalLock()
	GlobalUnlock()
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

func (t *mutableTree) GetImmutableAtHeight(version int64) (*ImmutableTree, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	tree, err := t.tree.GetImmutable(version)
	if err != nil {
		return nil, err
	}

	return &ImmutableTree{
		tree: tree,
	}, nil
}

func (t *mutableTree) GlobalLock() {
	t.Lock()
}

func (t *mutableTree) GlobalUnlock() {
	t.Unlock()
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

func (t *mutableTree) GetImmutable() *ImmutableTree {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return &ImmutableTree{
		tree: t.tree.ImmutableTree,
	}
}

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
func (t *mutableTree) DeleteVersionsFromInterval(fromVersion, toVersion int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	err := t.tree.DeleteVersionsFromInterval(fromVersion, toVersion)
	if err != nil {
		// mb error: unable to delete version %v with %v active readers
		// FIXME: edit architecture of using MutableTree and ImmutableTree
		if strings.HasPrefix(err.Error(), "unable to delete version ") && strings.HasSuffix(err.Error(), " active readers") {
			return nil
		}
		return err
	}
	return nil
}

// Should use GlobalLock() and GlobalUnlock
func (t *mutableTree) DeleteVersionIfExists(version int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if !t.tree.VersionExists(version) {
		return nil
	}

	return t.tree.DeleteVersion(version)
}

func (t *mutableTree) KeepLastHeight() int64 {
	t.lock.RLock()
	defer t.lock.RUnlock()

	versions := t.tree.AvailableVersions()
	prev := 1
	for _, version := range versions {
		if version-prev == 1 {
			break
		}
		prev = version
	}

	return int64(prev)
}

func (t *mutableTree) AvailableVersions() []int {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.AvailableVersions()
}

// ImmutableTree used for CheckState: API and CheckTx calls.
type ImmutableTree struct {
	tree *iavl.ImmutableTree
}

// NewImmutableTree returns MTree from given db at given height
// Warning: returns the MTree interface, but you should only use ReadOnlyTree
func NewImmutableTree(height uint64, db dbm.DB) (MTree, error) {
	tree, _ := NewMutableTree(0, db, 1024)
	_, err := tree.LazyLoadVersion(int64(height))
	if err != nil {
		return nil, err
	}
	return tree, nil
}

// Iterate iterates over all keys of the tree, in order. The keys and values must not be modified,
// since they may point to data stored within IAVL.
func (t *ImmutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	return t.tree.Iterate(fn)
}

// Hash returns the root hash.
func (t *ImmutableTree) Hash() []byte {
	return t.tree.Hash()
}

// Version returns the version of the tree.
func (t *ImmutableTree) Version() int64 {
	return t.tree.Version()
}

// Get returns the index and value of the specified key if it exists, or nil and the next index
// otherwise. The returned value must not be modified, since it may point to data stored within
// IAVL.
func (t *ImmutableTree) Get(key []byte) (index int64, value []byte) {
	return t.tree.Get(key)
}
