package tree

import (
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tm-db"
	"sync"
)

type Tree interface {
	Get(key []byte) (index int64, value []byte)
	Set(key, value []byte) bool
	Remove(key []byte) ([]byte, bool)
	LoadVersion(targetVersion int64) (int64, error)
	LazyLoadVersion(targetVersion int64) (int64, error)
	SaveVersion() ([]byte, int64, error)
	DeleteVersion(version int64) error
	GetImmutable() *ImmutableTree
	GetImmutableAtHeight(version int64) (*ImmutableTree, error)
	Version() int64
	Hash() []byte
	Iterate(fn func(key []byte, value []byte) bool) (stopped bool)
}

func NewMutableTree(db dbm.DB, cacheSize int) *MutableTree {
	tree, err := iavl.NewMutableTree(db, cacheSize)
	if err != nil {
		panic(err)
	}
	return &MutableTree{
		tree: tree,
	}
}

type MutableTree struct {
	tree *iavl.MutableTree

	lock sync.RWMutex
}

func (t *MutableTree) GetImmutableAtHeight(version int64) (*ImmutableTree, error) {
	tree, err := t.tree.GetImmutable(version)
	if err != nil {
		return nil, err
	}

	return &ImmutableTree{
		tree: tree,
	}, nil
}

func (t *MutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	return t.tree.Iterate(fn)
}

func (t *MutableTree) Hash() []byte {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Hash()
}

func (t *MutableTree) Version() int64 {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Version()
}

func (t *MutableTree) GetImmutable() *ImmutableTree {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return &ImmutableTree{
		tree: t.tree.ImmutableTree,
	}
}

func (t *MutableTree) Get(key []byte) (index int64, value []byte) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.tree.Get(key)
}

func (t *MutableTree) Set(key, value []byte) bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.Set(key, value)
}

func (t *MutableTree) Remove(key []byte) ([]byte, bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.Remove(key)
}

func (t *MutableTree) LoadVersion(targetVersion int64) (int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.LoadVersion(targetVersion)
}

func (t *MutableTree) LazyLoadVersion(targetVersion int64) (int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.LazyLoadVersion(targetVersion)
}

func (t *MutableTree) SaveVersion() ([]byte, int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.SaveVersion()
}

func (t *MutableTree) DeleteVersion(version int64) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.DeleteVersion(version)
}

func NewImmutableTree(db dbm.DB) *ImmutableTree {
	return &ImmutableTree{
		tree: iavl.NewImmutableTree(db, 1024),
	}
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

func (t *ImmutableTree) GetImmutable() *ImmutableTree {
	return t
}

func (t *ImmutableTree) Get(key []byte) (index int64, value []byte) {
	return t.tree.Get(key)
}

func (t *ImmutableTree) GetImmutableAtHeight(version int64) (*ImmutableTree, error) {
	panic("Not implemented")
}

func (t *ImmutableTree) Set(key, value []byte) bool {
	panic("Not implemented")
}

func (t *ImmutableTree) Remove(key []byte) ([]byte, bool) {
	panic("Not implemented")
}

func (t *ImmutableTree) LoadVersion(targetVersion int64) (int64, error) {
	panic("Not implemented")
}

func (t *ImmutableTree) LazyLoadVersion(targetVersion int64) (int64, error) {
	panic("Not implemented")
}

func (t *ImmutableTree) SaveVersion() ([]byte, int64, error) {
	panic("Not implemented")
}

func (t *ImmutableTree) DeleteVersion(version int64) error {
	panic("Not implemented")
}
