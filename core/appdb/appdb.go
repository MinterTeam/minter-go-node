package appdb

import (
	"encoding/binary"
	"errors"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tm-db"
)

var (
	cdc = amino.NewCodec()
)

const (
	hashPath           = "hash"
	heightPath         = "height"
	startHeightPath    = "startHeight"
	blockTimeDeltaPath = "blockDelta"
	blockTimeBeginPath = "blockBegin"
	blockTimePath      = "blockTime"
	validatorsPath     = "validators"

	dbName = "app"
)

type AppDB struct {
	db db.DB
}

func (appDB *AppDB) Close() {
	appDB.db.Close()
}

func (appDB *AppDB) GetLastBlockHash() []byte {
	var hash [32]byte

	rawHash, err := appDB.db.Get([]byte(hashPath))
	if err != nil {
		panic(err)
	}
	copy(hash[:], rawHash)

	return hash[:]
}

func (appDB *AppDB) SetLastBlockHash(hash []byte) {
	appDB.db.Set([]byte(hashPath), hash)
}

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

func (appDB *AppDB) SetLastHeight(height uint64) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	appDB.db.Set([]byte(heightPath), h)
}

func (appDB *AppDB) SetStartHeight(height uint64) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	appDB.db.Set([]byte(startHeightPath), h)
}

func (appDB *AppDB) GetStartHeight() uint64 {
	result, err := appDB.db.Get([]byte(startHeightPath))
	if err != nil {
		panic(err)
	}
	var height uint64

	if result != nil {
		height = binary.BigEndian.Uint64(result)
	}

	return height
}

func (appDB *AppDB) GetValidators() types.ValidatorUpdates {
	result, err := appDB.db.Get([]byte(validatorsPath))
	if err != nil {
		panic(err)
	}

	if len(result) == 0 {
		return types.ValidatorUpdates{}
	}

	var vals types.ValidatorUpdates

	err = cdc.UnmarshalBinaryBare(result, &vals)

	if err != nil {
		panic(err)
	}

	return vals
}

func (appDB *AppDB) SaveValidators(vals types.ValidatorUpdates) {
	data, err := cdc.MarshalBinaryBare(vals)

	if err != nil {
		panic(err)
	}

	appDB.db.Set([]byte(validatorsPath), data)
}

type LastBlocksTimeDelta struct {
	Height uint64
	Delta  int
}

func (appDB *AppDB) GetLastBlocksTimeDelta(height uint64) (int, error) {
	result, err := appDB.db.Get([]byte(blockTimeDeltaPath))
	if err != nil {
		panic(err)
	}
	if result == nil {
		return 0, errors.New("no info about LastBlocksTimeDelta is available")
	}

	data := LastBlocksTimeDelta{}
	err = cdc.UnmarshalBinaryBare(result, &data)
	if err != nil {
		panic(err)
	}

	if data.Height != height {
		return 0, errors.New("no info about LastBlocksTimeDelta is available")
	}

	return data.Delta, nil
}

func (appDB *AppDB) SetLastBlocksTimeDelta(height uint64, delta int) {
	data, err := cdc.MarshalBinaryBare(LastBlocksTimeDelta{
		Height: height,
		Delta:  delta,
	})

	if err != nil {
		panic(err)
	}

	appDB.db.Set([]byte(blockTimeDeltaPath), data)
}

func NewAppDB(cfg *config.Config) *AppDB {
	return &AppDB{
		db: db.NewDB(dbName, db.BackendType(cfg.DBBackend), utils.GetMinterHome()+"/data"),
	}
}
