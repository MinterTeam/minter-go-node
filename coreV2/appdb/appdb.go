package appdb

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/tree"
	"math/big"
	"sync"

	abcTypes "github.com/tendermint/tendermint/abci/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tm-db"
	"sync/atomic"
	"time"
)

const (
	pricePath       = "price"
	rewardPath      = "reward"
	emissionPath    = "emission"
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
	emission        *big.Int
	price           *TimePrice
	reward          *big.Int
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

func (appDB *AppDB) SetEmission(emission *big.Int) {
	appDB.WG.Wait()
	appDB.emission = emission
}
func (appDB *AppDB) SaveEmission() {
	if err := appDB.db.Set([]byte(emissionPath), appDB.emission.Bytes()); err != nil {
		panic(err)
	}
}

func (appDB *AppDB) Emission() (emission *big.Int) {
	// todo: mutex
	if appDB.emission == nil {
		result, err := appDB.db.Get([]byte(emissionPath))
		if err != nil {
			panic(err)
		}

		appDB.emission = big.NewInt(0).SetBytes(result)
	}
	return appDB.emission
}

type TimePrice struct {
	T      time.Time
	R0, R1 *big.Int
}

func (appDB *AppDB) SetPrice(t time.Time, r0, r1 *big.Int) {
	// calc new reward
	_, reserve0, reserve1 := appDB.GetPrice()
	// update data
	tp := &TimePrice{
		T:  t,
		R0: r0, // BIP
		R1: r1, // USDTE
	}
	appDB.price = tp

	var diff = big.NewInt(0)
	var reward *big.Int
	if big.NewRat(1, 1).SetFrac(r0, r1).Cmp(big.NewRat(1, 1).SetFrac(reserve0, reserve1)) == 1 {
		//100 / 10 = 10
		//50 / 20 = 2.5
		//цена выросла
		//10 / 2.5 = 4 раза
		diff = big.NewInt(0).Div(big.NewInt(0).Mul(reserve0, r1), big.NewInt(0).Mul(reserve1, r0))
		reward = big.NewInt(0).Add(appDB.GetReward(), diff)
	} else {
		// 100 / 10 = 10
		// 150 / 5 = 30
		// цена бип упала
		// 30 / 10 = 3
		diff = big.NewInt(0).Div(big.NewInt(0).Mul(reserve1, r0), big.NewInt(0).Mul(reserve0, r1))
		reward = big.NewInt(0).Sub(appDB.GetReward(), diff)
	}
	appDB.reward = reward

	panic(1)
}

func (appDB *AppDB) GetPrice() (t time.Time, r0, r1 *big.Int) {
	// todo: load
	if appDB.price == nil {
		return time.Time{}, nil, nil
	}
	return appDB.price.T, appDB.price.R0, appDB.price.R1
}

func (appDB *AppDB) GetReward() *big.Int {
	return appDB.reward
}
