package appdb

import (
	"bufio"
	"compress/zlib"
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/tree"
	"github.com/cosmos/cosmos-sdk/snapshots"
	snapshottypes "github.com/cosmos/cosmos-sdk/snapshots/types"
	"sync"

	"github.com/MinterTeam/minter-go-node/coreV2/appdb/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	iavltree "github.com/cosmos/iavl"
	protoio "github.com/gogo/protobuf/io"
	abcTypes "github.com/tendermint/tendermint/abci/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tm-db"
	"io"
	"math"
	"sync/atomic"
	"time"
)

const (
	hashPath        = "hash"
	heightPath      = "height"
	startHeightPath = "startHeight"
	blocksTimePath  = "blockDelta"
	validatorsPath  = "validators"
	versionsPath    = "versions"

	dbName = "app"
)

// AppDB is responsible for storing basic information about app state on disk
type AppDB struct {
	db db.DB
	//mu sync.RWMutex
	WG sync.WaitGroup

	store   tree.MTree
	stateDB db.DB

	startHeight    uint64
	lastHeight     uint64
	lastTimeBlocks []uint64
	validators     abciTypes.ValidatorUpdates

	isDirtyVersions bool
	versions        []*Version
}

// Close closes db connection, panics on error
func (appDB *AppDB) Close() error {
	if err := appDB.db.Close(); err != nil {
		return err
	}
	return nil
}

// GetLastBlockHash returns latest block hash stored on disk
func (appDB *AppDB) GetLastBlockHash() []byte {
	// appDB.mu.RLock()
	// defer appDB.mu.RUnlock()

	rawHash, err := appDB.db.Get([]byte(hashPath))
	if err != nil {
		panic(err)
	}

	if len(rawHash) == 0 {
		return nil
	}

	var hash [32]byte
	copy(hash[:], rawHash)
	return hash[:]
}

// SetLastBlockHash stores given block hash on disk, panics on error
func (appDB *AppDB) SetLastBlockHash(hash []byte) {

	appDB.WG.Wait()
	// appDB.mu.Lock()
	// defer appDB.mu.Unlock()

	if err := appDB.db.Set([]byte(hashPath), hash); err != nil {
		panic(err)
	}
}

// GetLastHeight returns latest block height stored on disk
func (appDB *AppDB) GetLastHeight() uint64 {
	// appDB.mu.RLock()
	// defer appDB.mu.RUnlock()
	return appDB.getLastHeight()
}
func (appDB *AppDB) getLastHeight() uint64 {
	val := atomic.LoadUint64(&appDB.lastHeight)
	if val != 0 {
		return val
	}

	result, err := appDB.db.Get([]byte(heightPath))
	if err != nil {
		panic(err)
	}

	if len(result) != 0 {
		val = binary.BigEndian.Uint64(result)
		atomic.StoreUint64(&appDB.lastHeight, val)
	}

	return val
}

// SetLastHeight stores given block height on disk, panics on error
func (appDB *AppDB) SetLastHeight(height uint64) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)

	appDB.WG.Wait()
	// appDB.mu.Lock()
	// defer appDB.mu.Unlock()

	if err := appDB.db.Set([]byte(heightPath), h); err != nil {
		panic(err)
	}

	atomic.StoreUint64(&appDB.lastHeight, height)
}

// SetStartHeight stores given block height on disk as start height, panics on error
func (appDB *AppDB) SetStartHeight(height uint64) {
	atomic.StoreUint64(&appDB.startHeight, height)
}

// SaveStartHeight stores given block height on disk as start height, panics on error
func (appDB *AppDB) SaveStartHeight() {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, atomic.LoadUint64(&appDB.startHeight))

	appDB.WG.Wait()
	// appDB.mu.Lock()
	// defer appDB.mu.Unlock()

	if err := appDB.db.Set([]byte(startHeightPath), h); err != nil {
		panic(err)
	}
}

// GetStartHeight returns start height stored on disk
func (appDB *AppDB) GetStartHeight() uint64 {
	val := atomic.LoadUint64(&appDB.startHeight)
	if val != 0 {
		return val
	}

	// appDB.mu.RLock()
	// defer appDB.mu.RUnlock()

	result, err := appDB.db.Get([]byte(startHeightPath))
	if err != nil {
		panic(err)
	}

	if len(result) != 0 {
		val = binary.BigEndian.Uint64(result)
		atomic.StoreUint64(&appDB.startHeight, val)
	}

	return val
}

// GetValidators returns list of latest validators stored on dist
func (appDB *AppDB) GetValidators() abcTypes.ValidatorUpdates {
	if appDB.validators != nil {
		return appDB.validators
	}

	// appDB.mu.RLock()
	// defer appDB.mu.RUnlock()

	result, err := appDB.db.Get([]byte(validatorsPath))
	if err != nil {
		panic(err)
	}

	if len(result) == 0 {
		return abcTypes.ValidatorUpdates{}
	}

	var vals abcTypes.ValidatorUpdates

	err = tmjson.Unmarshal(result, &vals)
	if err != nil {
		panic(err)
	}

	return vals
}

// SetValidators sets given validators list on mem
func (appDB *AppDB) SetValidators(vals abcTypes.ValidatorUpdates) {
	appDB.validators = vals
}

// FlushValidators stores validators list from mem to disk, panics on error
func (appDB *AppDB) FlushValidators() {
	if appDB.validators == nil {
		return
	}
	data, err := tmjson.Marshal(appDB.validators)
	if err != nil {
		panic(err)
	}

	appDB.WG.Wait()
	// appDB.mu.Lock()
	// defer appDB.mu.Unlock()

	if err := appDB.db.Set([]byte(validatorsPath), data); err != nil {
		panic(err)
	}
	appDB.validators = nil
}

const BlocksTimeCount = 4

// GetLastBlockTimeDelta returns delta of time between latest blocks
func (appDB *AppDB) GetLastBlockTimeDelta() (sumTimes int, count int) {
	if len(appDB.lastTimeBlocks) == 0 {

		// appDB.mu.RLock()
		// defer appDB.mu.RUnlock()

		result, err := appDB.db.Get([]byte(blocksTimePath))
		if err != nil {
			panic(err)
		}

		if len(result) == 0 {
			return 0, 0
		}

		err = tmjson.Unmarshal(result, &appDB.lastTimeBlocks)
		if err != nil {
			panic(err)
		}
	}

	return calcBlockDelta(appDB.lastTimeBlocks)
}

func calcBlockDelta(times []uint64) (sumTimes int, num int) {
	count := len(times)
	if count < 2 {
		return 0, count - 1
	}

	var res int
	for i, timestamp := range times[1:] {
		res += int(timestamp - times[i])
	}
	return res, count - 1
}

func (appDB *AppDB) AddBlocksTime(time time.Time) {
	if len(appDB.lastTimeBlocks) == 0 {

		// appDB.mu.RLock()
		// defer appDB.mu.RUnlock()

		result, err := appDB.db.Get([]byte(blocksTimePath))
		if err != nil {
			panic(err)
		}
		if len(result) != 0 {
			err = tmjson.Unmarshal(result, &appDB.lastTimeBlocks)
			if err != nil {
				panic(err)
			}
		}
	}

	appDB.lastTimeBlocks = append(appDB.lastTimeBlocks, uint64(time.Unix()))
	count := len(appDB.lastTimeBlocks)
	if count > BlocksTimeCount {
		appDB.lastTimeBlocks = appDB.lastTimeBlocks[count-BlocksTimeCount:]
	}
}

func (appDB *AppDB) SaveBlocksTime() {
	data, err := tmjson.Marshal(appDB.lastTimeBlocks)
	if err != nil {
		panic(err)
	}

	appDB.WG.Wait()
	// appDB.mu.Lock()
	// defer appDB.mu.Unlock()

	if err := appDB.db.Set([]byte(blocksTimePath), data); err != nil {
		panic(err)
	}
}

type Version struct {
	Name   string
	Height uint64
}

func (appDB *AppDB) GetVersionName(height uint64) string {
	lastVersionName := ""
	for _, version := range appDB.GetVersions() {
		if version.Height > height {
			return lastVersionName
		}
		lastVersionName = version.Name
	}

	return lastVersionName
}

func (appDB *AppDB) GetVersionHeight(name string) uint64 {
	for _, version := range appDB.GetVersions() {
		if version.Name == name {
			return version.Height
		}
	}

	return 0
}

func (appDB *AppDB) GetVersions() []*Version {
	if len(appDB.versions) == 0 {

		// appDB.mu.RLock()
		// defer appDB.mu.RUnlock()

		result, err := appDB.db.Get([]byte(versionsPath))
		if err != nil {
			panic(err)
		}
		if len(result) != 0 {
			err = tmjson.Unmarshal(result, &appDB.versions)
			if err != nil {
				panic(err)
			}
		}
	}

	return appDB.versions
}

func (appDB *AppDB) AddVersion(v string, height uint64) {
	appDB.GetVersions()

	elem := &Version{
		Name:   v,
		Height: height,
	}
	// appDB.version = elem
	appDB.versions = append(appDB.versions, elem)
	appDB.isDirtyVersions = true
}

func (appDB *AppDB) SaveVersions() {
	if !appDB.isDirtyVersions {
		return
	}
	data, err := tmjson.Marshal(appDB.versions)
	if err != nil {
		panic(err)
	}

	appDB.WG.Wait()
	// appDB.mu.Lock()
	// defer appDB.mu.Unlock()

	if err := appDB.db.Set([]byte(versionsPath), data); err != nil {
		panic(err)
	}

	appDB.isDirtyVersions = false
}

func (appDB *AppDB) SetState(state tree.MTree) {
	appDB.store = state
}
func (appDB *AppDB) SetStateDB(stateDB db.DB) {
	appDB.stateDB = stateDB
}

// NewAppDB creates AppDB instance with given config
func NewAppDB(homeDir string, cfg *config.Config) *AppDB {
	newDB, err := db.NewDB(dbName, db.BackendType(cfg.DBBackend), homeDir+"/data")
	if err != nil {
		panic(err)
	}
	return &AppDB{
		db: newDB,
	}
}

//---------------------- Snapshotting ------------------

const (
	// Do not change chunk size without new snapshot format (must be uniform across nodes)
	snapshotChunkSize   = uint64(10e6)
	snapshotBufferSize  = int(snapshotChunkSize)
	snapshotMaxItemSize = int(64e6) // SDK has no key/value size limit, so we set an arbitrary limit
)

// Snapshot implements snapshottypes.Snapshotter. The snapshot output for a given format must be
// identical across nodes such that chunks from different sources fit together. If the output for a
// given format changes (at the byte level), the snapshot format must be bumped - see
// TestMultistoreSnapshot_Checksum test.
func (appDB *AppDB) Snapshot(height uint64, format uint32) (<-chan io.ReadCloser, error) {
	if format != snapshottypes.CurrentFormat {
		appDB.WG.Done()
		return nil, sdkerrors.Wrapf(snapshottypes.ErrUnknownFormat, "format %v", format)
	}

	var results []*types.SnapshotItem
	if height != appDB.GetLastHeight() {
		appDB.WG.Done()
		return nil, sdkerrors.Wrapf(sdkerrors.ErrLogic, "cannot snapshot future height %v", height)
	}

	if height == 0 {
		appDB.WG.Done()
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "cannot snapshot height 0")
	}

	for _, name := range []string{validatorsPath, heightPath, hashPath, versionsPath, blocksTimePath, startHeightPath} {
		result, err := appDB.db.Get([]byte(name))
		if err != nil {
			panic(err)
		}

		results = append(results, &types.SnapshotItem{
			Item: &types.SnapshotItem_Store{
				Store: &types.SnapshotStoreItem{
					Name:  name,
					Value: result,
				},
			},
		})
	}
	appDB.WG.Done()

	// Spawn goroutine to generate snapshot chunks and pass their io.ReadClosers through a channel
	ch := make(chan io.ReadCloser)
	go func() {
		// Set up a stream pipeline to serialize snapshot nodes:
		// ExportNode -> delimited Protobuf -> zlib -> buffer -> chunkWriter -> chan io.ReadCloser
		chunkWriter := snapshots.NewChunkWriter(ch, snapshotChunkSize)
		defer chunkWriter.Close()
		bufWriter := bufio.NewWriterSize(chunkWriter, snapshotBufferSize)
		defer func() {
			if err := bufWriter.Flush(); err != nil {
				chunkWriter.CloseWithError(err)
			}
		}()
		zWriter, err := zlib.NewWriterLevel(bufWriter, 7)
		if err != nil {
			chunkWriter.CloseWithError(sdkerrors.Wrap(err, "zlib failure"))
			return
		}
		defer func() {
			if err := zWriter.Close(); err != nil {
				chunkWriter.CloseWithError(err)
			}
		}()
		protoWriter := protoio.NewDelimitedWriter(zWriter)
		defer func() {
			if err := protoWriter.Close(); err != nil {
				chunkWriter.CloseWithError(err)
			}
		}()

		{
			for _, s := range results {
				err = protoWriter.WriteMsg(s)
				if err != nil {
					chunkWriter.CloseWithError(err)
					return
				}
			}

			// Export each IAVL store. Stores are serialized as a stream of SnapshotItem Protobuf
			// messages. The first item contains a SnapshotStore with store metadata (i.e. name),
			// and the following messages contain a SnapshotNode (i.e. an ExportNode). Store changes
			// are demarcated by new SnapshotStore items.

			exporter, err := appDB.store.Export(int64(height))
			if err != nil {
				chunkWriter.CloseWithError(err)
				return
			}
			defer exporter.Close()
			err = protoWriter.WriteMsg(&types.SnapshotItem{
				Item: &types.SnapshotItem_Store{
					Store: &types.SnapshotStoreItem{
						Name: "state",
					},
				},
			})
			if err != nil {
				chunkWriter.CloseWithError(err)
				return
			}

			for {
				node, err := exporter.Next()
				if err == iavltree.ExportDone {
					break
				} else if err != nil {
					chunkWriter.CloseWithError(err)
					return
				}
				err = protoWriter.WriteMsg(&types.SnapshotItem{
					Item: &types.SnapshotItem_IAVL{
						IAVL: &types.SnapshotIAVLItem{
							Key:     node.Key,
							Value:   node.Value,
							Height:  int32(node.Height),
							Version: node.Version,
						},
					},
				})
				if err != nil {
					chunkWriter.CloseWithError(err)
					return
				}
			}
			exporter.Close()
		}
	}()

	return ch, nil
}

// Restore implements snapshottypes.Snapshotter.
func (appDB *AppDB) Restore(
	height uint64, format uint32, chunks <-chan io.ReadCloser, ready chan<- struct{},
) error {
	if format != snapshottypes.CurrentFormat {
		return sdkerrors.Wrapf(snapshottypes.ErrUnknownFormat, "format %v", format)
	}
	if height == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrLogic, "cannot restore snapshot at height 0")
	}
	if height > uint64(math.MaxInt64) {
		return sdkerrors.Wrapf(snapshottypes.ErrInvalidMetadata,
			"snapshot height %v cannot exceed %v", height, int64(math.MaxInt64))
	}

	// Signal readiness. Must be done before the readers below are set up, since the zlib
	// reader reads from the stream on initialization, potentially causing deadlocks.
	if ready != nil {
		close(ready)
	}

	// Set up a restore stream pipeline
	// chan io.ReadCloser -> chunkReader -> zlib -> delimited Protobuf -> ExportNode
	chunkReader := snapshots.NewChunkReader(chunks)
	defer chunkReader.Close()
	zReader, err := zlib.NewReader(chunkReader)
	if err != nil {
		return sdkerrors.Wrap(err, "zlib failure")
	}
	defer zReader.Close()
	protoReader := protoio.NewDelimitedReader(zReader, snapshotMaxItemSize)
	defer protoReader.Close()

	// Import nodes into stores. The first item is expected to be a SnapshotItem containing
	// a SnapshotStoreItem, telling us which store to import into. The following items will contain
	// SnapshotNodeItem (i.e. ExportNode) until we reach the next SnapshotStoreItem or EOF.
	var importer *iavltree.Importer
	for {
		item := &types.SnapshotItem{}
		err := protoReader.ReadMsg(item)
		if err == io.EOF {
			break
		} else if err != nil {
			return sdkerrors.Wrap(err, "invalid protobuf message")
		}

		switch item := item.Item.(type) {
		case *types.SnapshotItem_Store:
			switch item.Store.Name {
			case "state":
				if importer != nil {
					err = importer.Commit()
					if err != nil {
						return sdkerrors.Wrap(err, "IAVL commit failed")
					}
					importer.Close()
				}
				if appDB.store == nil {
					appDB.store, err = tree.NewMutableTree(0, appDB.stateDB, 1000000, appDB.GetStartHeight())
					if err != nil {
						return sdkerrors.Wrap(err, "create state failed")
					}
				}
				importer, err = appDB.store.Import(int64(height))
				if err != nil {
					return sdkerrors.Wrap(err, "import failed")
				}
				defer importer.Close()

			case validatorsPath, heightPath, hashPath, versionsPath, blocksTimePath, startHeightPath:
				if err := appDB.db.Set([]byte(item.Store.Name), item.Store.Value); err != nil {
					panic(err)
				}

			default:
				return sdkerrors.Wrapf(sdkerrors.ErrLogic, "unknown store name %v", item.Store.Name)
			}
		case *types.SnapshotItem_IAVL:
			if importer == nil {
				return sdkerrors.Wrap(sdkerrors.ErrLogic, "received IAVL node item before store item")
			}
			if item.IAVL.Height > math.MaxInt8 {
				return sdkerrors.Wrapf(sdkerrors.ErrLogic, "node height %v cannot exceed %v",
					item.IAVL.Height, math.MaxInt8)
			}
			node := &iavltree.ExportNode{
				Key:     item.IAVL.Key,
				Value:   item.IAVL.Value,
				Height:  int8(item.IAVL.Height),
				Version: item.IAVL.Version,
			}
			// Protobuf does not differentiate between []byte{} as nil, but fortunately IAVL does
			// not allow nil keys nor nil values for leaf nodes, so we can always set them to empty.
			if node.Key == nil {
				node.Key = []byte{}
			}
			if node.Height == 0 && node.Value == nil {
				node.Value = []byte{}
			}
			err := importer.Add(node)
			if err != nil {
				return sdkerrors.Wrap(err, "IAVL node import failed")
			}

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrLogic, "unknown snapshot item %T", item)
		}
	}

	if importer != nil {
		err := importer.Commit()
		if err != nil {
			return sdkerrors.Wrap(err, "IAVL commit failed")
		}
		importer.Close()
	}

	return nil
}
