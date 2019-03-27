package state

import (
	"github.com/danil-lashin/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
	"sync"
)

type Tree interface {
	Get(key []byte) (index int64, value []byte)
	Set(key, value []byte) bool
	Remove(key []byte) ([]byte, bool)
	LoadVersion(targetVersion int64) (int64, error)
	Load() (int64, error)
	SaveVersion() ([]byte, int64, error)
	DeleteVersion(version int64) error
	GetImmutable() *ImmutableTree
	Version() int64
	Hash() []byte
	Iterate(fn func(key []byte, value []byte) bool) (stopped bool)
}

func NewMutableTree(db dbm.DB) *MutableTree {
	return &MutableTree{
		tree: iavl.NewMutableTree(db, 1024),
	}
}

type MutableTree struct {
	tree *iavl.MutableTree

	lock sync.RWMutex
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

func (t *MutableTree) Load() (int64, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.tree.Load()
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

func (t *ImmutableTree) Load() (int64, error) {
	panic("Not implemented")
}

func (t *ImmutableTree) GetImmutable() *ImmutableTree {
	return t
}

func (t *ImmutableTree) Get(key []byte) (index int64, value []byte) {
	return t.tree.Get(key)
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

func (t *ImmutableTree) SaveVersion() ([]byte, int64, error) {
	panic("Not implemented")
}

func (t *ImmutableTree) DeleteVersion(version int64) error {
	panic("Not implemented")
}
