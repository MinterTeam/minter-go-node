package appdb

import (
	"encoding/binary"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/math"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/MinterTeam/minter-go-node/tree"
	db "github.com/tendermint/tm-db"
	"math/big"
	"sync"

	abcTypes "github.com/tendermint/tendermint/abci/types"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"sync/atomic"
	"time"
)

const (
	pricePath       = "price"
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
	WG sync.WaitGroup
	mu sync.Mutex

	store   tree.MTree
	stateDB db.DB

	startHeight    uint64
	lastHeight     uint64
	lastTimeBlocks []uint64
	validators     abciTypes.ValidatorUpdates

	isDirtyVersions bool
	versions        []*Version

	isDirtyEmission bool
	emission        *big.Int

	isDirtyPrice bool
	price        *TimePrice
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
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	// todo add field hash

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

	if err := appDB.db.Set([]byte(hashPath), hash); err != nil {
		panic(err)
	}
}

// GetLastHeight returns latest block height stored on disk
func (appDB *AppDB) GetLastHeight() uint64 {
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
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if appDB.validators != nil {
		return appDB.validators
	}

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
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	appDB.validators = vals
}

// FlushValidators stores validators list from mem to disk, panics on error
func (appDB *AppDB) FlushValidators() {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if appDB.validators == nil {
		return
	}
	data, err := tmjson.Marshal(appDB.validators)
	if err != nil {
		panic(err)
	}

	appDB.WG.Wait()

	if err := appDB.db.Set([]byte(validatorsPath), data); err != nil {
		panic(err)
	}
	appDB.validators = nil
}

const BlocksTimeCount = 4

// GetLastBlockTimeDelta returns delta of time between latest blocks
func (appDB *AppDB) GetLastBlockTimeDelta() (sumTimes int, count int) {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if len(appDB.lastTimeBlocks) == 0 {
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
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if len(appDB.lastTimeBlocks) == 0 {
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
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	data, err := tmjson.Marshal(appDB.lastTimeBlocks)
	if err != nil {
		panic(err)
	}

	appDB.WG.Wait()

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
			return version.Height + 1
		}
	}

	return 0
}

func (appDB *AppDB) GetVersions() []*Version {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if len(appDB.versions) == 0 {
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

	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	appDB.versions = append(appDB.versions, elem)
	appDB.isDirtyVersions = true
}

func (appDB *AppDB) SaveVersions() {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if !appDB.isDirtyVersions {
		return
	}
	data, err := tmjson.Marshal(appDB.versions)
	if err != nil {
		panic(err)
	}

	appDB.WG.Wait()

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
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	appDB.isDirtyEmission = true
	appDB.emission = emission
}

func (appDB *AppDB) SaveEmission() {
	if appDB.isDirtyEmission == false {
		return
	}
	appDB.isDirtyEmission = false

	appDB.WG.Wait()
	if err := appDB.db.Set([]byte(emissionPath), appDB.emission.Bytes()); err != nil {
		panic(err)
	}
}

func (appDB *AppDB) Emission() (emission *big.Int) {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if appDB.emission == nil {
		result, err := appDB.db.Get([]byte(emissionPath))
		if err != nil {
			panic(err)
		}

		if len(result) == 0 {
			return nil
		}

		appDB.emission = big.NewInt(0).SetBytes(result)
	}
	return appDB.emission
}

type TimePrice struct {
	T      uint64
	R0, R1 *big.Int
	Off    bool
	Last   *big.Int
}

func (appDB *AppDB) UpdatePriceFix(t time.Time, r0, r1 *big.Int) (reward, safeReward *big.Int) {
	tOld, reserve0, reserve1, last, off := appDB.GetPrice()

	fNew := big.NewRat(1, 1).SetFrac(r1, r0)
	// Price ^ (1/4) * 350
	priceCount, _ := new(big.Float).Mul(new(big.Float).Mul(math.Pow(new(big.Float).SetRat(fNew), big.NewFloat(0.25)), big.NewFloat(350)), big.NewFloat(1e18)).Int(nil)
	if tOld.IsZero() {
		appDB.SetPrice(t, r0, r1, priceCount, false)
		return new(big.Int).Set(priceCount), new(big.Int).Set(priceCount)
	}

	defer func() { appDB.SetPrice(t, r0, r1, last, off) }()

	fOld := big.NewRat(1, 1).SetFrac(reserve1, reserve0)

	rat := new(big.Rat).Mul(new(big.Rat).Quo(new(big.Rat).Sub(fNew, fOld), fOld), new(big.Rat).SetInt64(100))
	diff := big.NewInt(0).Div(rat.Num(), rat.Denom())

	if diff.Cmp(big.NewInt(-10)) != 1 {
		last.SetInt64(0)
		off = true
		return last, new(big.Int).Set(priceCount)
	}

	if off && last.Cmp(priceCount) == -1 {
		last.Add(last, big.NewInt(5e18))
		last.Add(last, big.NewInt(5e18))
		burn := big.NewInt(0).Sub(priceCount, last)
		if burn.Sign() != 1 {
			last.Set(priceCount)
			off = false
			return new(big.Int).Set(last), new(big.Int).Set(priceCount)
		}
		return new(big.Int).Set(last), new(big.Int).Set(priceCount)
	}

	off = false
	last.Set(priceCount)

	return new(big.Int).Set(last), new(big.Int).Set(priceCount)
}

// UpdatePriceBug
// Deprecated
func (appDB *AppDB) UpdatePriceBug(t time.Time, r0, r1 *big.Int) (reward, safeReward *big.Int) {
	tOld, reserve0, reserve1, last, off := appDB.GetPrice()

	fNew := big.NewRat(1, 1).SetFrac(r1, r0)
	// Price ^ (1/4) * 350
	priceCount, _ := new(big.Float).Mul(new(big.Float).Mul(math.Pow(new(big.Float).SetRat(fNew), big.NewFloat(0.25)), big.NewFloat(350)), big.NewFloat(1e18)).Int(nil)
	if tOld.IsZero() {
		appDB.SetPrice(t, r0, r1, priceCount, false)
		return new(big.Int).Set(priceCount), new(big.Int).Set(priceCount)
	}

	defer func() { appDB.SetPrice(t, r0, r1, last, off) }()

	fOld := big.NewRat(1, 1).SetFrac(reserve1, reserve0)

	rat := new(big.Rat).Mul(new(big.Rat).Quo(new(big.Rat).Sub(fNew, fOld), fOld), new(big.Rat).SetInt64(100))
	diff := big.NewInt(0).Div(rat.Num(), rat.Denom())

	if diff.Cmp(big.NewInt(-10)) != 1 {
		last.SetInt64(0)
		off = true
		return last, new(big.Int).Set(priceCount)
	}

	if off && diff.Sign() != -1 {
		last.Add(last, big.NewInt(5e18))
		last.Add(last, big.NewInt(5e18))
		burn := big.NewInt(0).Sub(priceCount, last)
		if burn.Sign() != 1 {
			last.Set(priceCount)
			off = false
			return new(big.Int).Set(last), new(big.Int).Set(priceCount)
		}
		return new(big.Int).Set(last), new(big.Int).Set(priceCount)
	}

	off = false
	last.Set(priceCount)

	return new(big.Int).Set(last), new(big.Int).Set(priceCount)
}

func (appDB *AppDB) SetPrice(t time.Time, r0, r1 *big.Int, lastReward *big.Int, off bool) {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	appDB.price = &TimePrice{
		T:    uint64(t.UTC().UnixNano()),
		R0:   big.NewInt(0).Set(r0), // BIP
		R1:   big.NewInt(0).Set(r1), // USDTE
		Off:  off,
		Last: new(big.Int).Set(lastReward),
	}
	appDB.isDirtyPrice = true
}
func (appDB *AppDB) SavePrice() {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if appDB.isDirtyPrice == false {
		return
	}
	appDB.isDirtyPrice = false

	appDB.WG.Wait()

	bytes, err := rlp.EncodeToBytes(appDB.price)
	if err != nil {
		panic(err)
	}

	err = appDB.db.Set([]byte(pricePath), bytes)
	if err != nil {
		panic(err)
	}
}

func (appDB *AppDB) GetPrice() (t time.Time, r0, r1 *big.Int, lastReward *big.Int, off bool) {
	appDB.mu.Lock()
	defer appDB.mu.Unlock()

	if appDB.price == nil {
		result, err := appDB.db.Get([]byte(pricePath))
		if err != nil {
			panic(err)
		}
		if len(result) == 0 {
			return time.Time{}, nil, nil, nil, false
		}

		appDB.price = &TimePrice{}

		err = rlp.DecodeBytes(result, appDB.price)
		if err != nil {
			panic(err)
		}
	}
	return time.Unix(0, int64(appDB.price.T)).UTC(), appDB.price.R0, appDB.price.R1, new(big.Int).Set(appDB.price.Last), appDB.price.Off
}
