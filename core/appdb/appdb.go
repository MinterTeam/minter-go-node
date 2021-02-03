package appdb

import (
	"encoding/binary"
	"errors"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/tendermint/abci/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tm-db"
)

const (
	hashPath           = "hash"
	heightPath         = "height"
	startHeightPath    = "startHeight"
	blockTimeDeltaPath = "blockDelta"
	validatorsPath     = "validators"

	dbName = "app"
)

func init() {
	tmjson.RegisterType(&lastBlocksTimeDelta{}, "last_blocks_time_delta")
}

// AppDB is responsible for storing basic information about app state on disk
type AppDB struct {
	db          db.DB
	startHeight uint64
	validators  abciTypes.ValidatorUpdates
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
	rawHash, err := appDB.db.Get([]byte(hashPath))
	if err != nil {
		panic(err)
	}

	var hash [32]byte
	copy(hash[:], rawHash)

	return hash[:]
}

// SetLastBlockHash stores given block hash on disk, panics on error
func (appDB *AppDB) SetLastBlockHash(hash []byte) {
	if err := appDB.db.Set([]byte(hashPath), hash); err != nil {
		panic(err)
	}
}

// GetLastHeight returns latest block height stored on disk
func (appDB *AppDB) GetLastHeight() uint64 {
	result, err := appDB.db.Get([]byte(heightPath))
	if err != nil {
		panic(err)
	}
	var height uint64

	if result != nil {
		height = binary.BigEndian.Uint64(result)
	}

	return height
}

// SetLastHeight stores given block height on disk, panics on error
func (appDB *AppDB) SetLastHeight(height uint64) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	if err := appDB.db.Set([]byte(heightPath), h); err != nil {
		panic(err)
	}
}

// SetStartHeight stores given block height on disk as start height, panics on error
func (appDB *AppDB) SetStartHeight(height uint64) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	if err := appDB.db.Set([]byte(startHeightPath), h); err != nil {
		panic(err)
	}
	appDB.startHeight = height
}

// GetStartHeight returns start height stored on disk
func (appDB *AppDB) GetStartHeight() uint64 {
	if appDB.startHeight != 0 {
		return appDB.startHeight
	}
	result, err := appDB.db.Get([]byte(startHeightPath))
	if err != nil {
		panic(err)
	}

	if result != nil {
		appDB.startHeight = binary.BigEndian.Uint64(result)
	}

	return appDB.startHeight
}

// GetValidators returns list of latest validators stored on dist
func (appDB *AppDB) GetValidators() types.ValidatorUpdates {
	if appDB.validators != nil {
		return appDB.validators
	}

	result, err := appDB.db.Get([]byte(validatorsPath))
	if err != nil {
		panic(err)
	}

	if len(result) == 0 {
		return types.ValidatorUpdates{}
	}

	var vals types.ValidatorUpdates

	err = tmjson.Unmarshal(result, &vals)
	if err != nil {
		panic(err)
	}

	return vals
}

// SetValidators sets given validators list on mem
func (appDB *AppDB) SetValidators(vals types.ValidatorUpdates) {
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

	if err := appDB.db.Set([]byte(validatorsPath), data); err != nil {
		panic(err)
	}
	appDB.validators = nil
}

type lastBlocksTimeDelta struct {
	Height uint64
	Delta  int
}

// GetLastBlocksTimeDelta returns delta of time between latest blocks
func (appDB *AppDB) GetLastBlocksTimeDelta(height uint64) (int, error) {
	result, err := appDB.db.Get([]byte(blockTimeDeltaPath))
	if err != nil {
		panic(err)
	}
	if result == nil {
		return 0, errors.New("no info about lastBlocksTimeDelta is available")
	}

	data := lastBlocksTimeDelta{}
	err = tmjson.Unmarshal(result, &data)
	if err != nil {
		panic(err)
	}

	if data.Height != height {
		return 0, errors.New("no info about lastBlocksTimeDelta is available")
	}

	return data.Delta, nil
}

// SetLastBlocksTimeDelta stores delta of time between latest blocks
func (appDB *AppDB) SetLastBlocksTimeDelta(height uint64, delta int) {
	data, err := tmjson.Marshal(&lastBlocksTimeDelta{
		Height: height,
		Delta:  delta,
	})

	if err != nil {
		panic(err)
	}

	if err := appDB.db.Set([]byte(blockTimeDeltaPath), data); err != nil {
		panic(err)
	}
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
