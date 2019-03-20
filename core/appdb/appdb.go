package appdb

import (
	"encoding/binary"
	"errors"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
)

var (
	cdc = amino.NewCodec()
)

const (
	hashPath           = "hash"
	heightPath         = "height"
	blockTimeDeltaPath = "blockDelta"
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

	rawHash := appDB.db.Get([]byte(hashPath))
	copy(hash[:], rawHash)

	return hash[:]
}

func (appDB *AppDB) SetLastBlockHash(hash []byte) {
	appDB.db.Set([]byte(hashPath), hash)
}

func (appDB *AppDB) GetLastHeight() uint64 {
	result := appDB.db.Get([]byte(heightPath))
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

func (appDB *AppDB) GetValidators() types.ValidatorUpdates {
	result := appDB.db.Get([]byte(validatorsPath))

	if len(result) == 0 {
		return types.ValidatorUpdates{}
	}

	var vals types.ValidatorUpdates

	err := cdc.UnmarshalBinaryBare(result, &vals)

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
	result := appDB.db.Get([]byte(blockTimeDeltaPath))
	if result == nil {
		return 0, errors.New("no info about LastBlocksTimeDelta is available")
	}

	data := LastBlocksTimeDelta{}
	err := cdc.UnmarshalBinaryBare(result, &data)
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

func NewAppDB() *AppDB {
	ldb, err := db.NewGoLevelDB(dbName, utils.GetMinterHome()+"/data")

	if err != nil {
		panic(err)
	}

	return &AppDB{
		db: ldb,
	}
}
