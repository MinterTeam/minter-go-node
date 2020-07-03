package tree

import (
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"sync"
)

type ReadOnlyTree interface {
	Get(key []byte) (index int64, value []byte)
	Version() int64
	Hash() []byte
	Iterate(fn func(key []byte, value []byte) bool) (stopped bool)
	KeepLastHeight() int64
	AvailableVersions() []int
}

type MTree interface {
	ReadOnlyTree
	Set(key, value []byte) bool
	Remove(key []byte) ([]byte, bool)
	LoadVersion(targetVersion int64) (int64, error)
	LazyLoadVersion(targetVersion int64) (int64, error)
	SaveVersion() ([]byte, int64, error)
	DeleteVersion(version int64) error
	GetImmutable() *ImmutableTree
	GetImmutableAtHeight(version int64) (*ImmutableTree, error)
}

func NewMutableTree(height uint64, db dbm.DB, cacheSize int) MTree {
	tree, err := iavl.NewMutableTree(db, cacheSize)
	if err != nil {
		panic(err)
	}

	_, err = tree.LoadVersionForOverwriting(int64(height))
	if err != nil {
		panic(err)
	}

	return &mutableTree{
		tree: tree,
	}
}

type mutableTree struct {
	tree *iavl.MutableTree
	lock sync.RWMutex
}

func (t *mutableTree) GetImmutableAtHeight(version int64) (*ImmutableTree, error) {
	tree, err := t.tree.GetImmutable(version)
	if err != nil {
		return nil, err
	}

	return &ImmutableTree{
		tree: tree,
	}, nil
}

func (t *mutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
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

func (t *mutableTree) SaveVersion() ([]byte, int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.SaveVersion()
}

func (t *mutableTree) DeleteVersion(version int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.DeleteVersion(version)
}

func (t *mutableTree) KeepLastHeight() int64 {
	t.lock.Lock()
	defer t.lock.Unlock()

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
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.AvailableVersions()
}

type ImmutableTree struct {
	tree *iavl.ImmutableTree
}

func (t *ImmutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	return t.tree.Iterate(fn)
}

func (t *ImmutableTree) Hash() []byte {
	return t.tree.Hash()
}

func (t *ImmutableTree) Version() int64 {
	return t.tree.Version()
}

func (t *ImmutableTree) Get(key []byte) (index int64, value []byte) {
	return t.tree.Get(key)
}
