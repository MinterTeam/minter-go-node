package appdb

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
)

var (
	cdc = amino.NewCodec()
)

const (
	hashPath       = "hash"
	heightPath     = "height"
	validatorsPath = "validators"

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

func (appDB *AppDB) GetLastHeight() int64 {
	result := appDB.db.Get([]byte(heightPath))
	var height int64 = 0

	if result != nil {
		height = int64(binary.BigEndian.Uint64(result))
	}

	return height
}

func (appDB *AppDB) SetLastHeight(height int64) {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, uint64(height))
	appDB.db.Set([]byte(heightPath), h)
}

func (appDB *AppDB) GetValidators() types.ValidatorUpdates {
	result := appDB.db.Get([]byte(validatorsPath))

	if len(result) == 0 {
		return types.ValidatorUpdates{}
	}

	var vals types.ValidatorUpdates

	err := cdc.UnmarshalBinaryLengthPrefixed(result, &vals)

	if err != nil {
		panic(err)
	}

	return vals
}

func (appDB *AppDB) SaveValidators(vals types.ValidatorUpdates) {
	data, err := cdc.MarshalBinaryLengthPrefixed(vals)

	if err != nil {
		panic(err)
	}

	appDB.db.Set([]byte(validatorsPath), data)
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
