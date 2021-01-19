package minter

import (
	"bytes"
	"fmt"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/appdb"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/state/candidates"
	"github.com/MinterTeam/minter-go-node/core/statistics"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/MinterTeam/minter-go-node/version"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/go-amino"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmNode "github.com/tendermint/tendermint/node"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Statuses of validators
const (
	ValidatorPresent = 1
	ValidatorAbsent  = 2
)

// Block params
const (
	blockMaxBytes = 10000000
	defaultMaxGas = 100000
	minMaxGas     = 5000
)

const votingPowerConsensus = 2. / 3.

// Blockchain is a main structure of Minter
type Blockchain struct {
	abciTypes.BaseApplication

	statisticData *statistics.Data

	appDB              *appdb.AppDB
	eventsDB           eventsdb.IEventsDB
	stateDeliver       *state.State
	stateCheck         *state.CheckState
	height             uint64   // current Blockchain height
	rewards            *big.Int // Rewards pool
	validatorsStatuses map[types.TmAddress]int8

	// local rpc client for Tendermint
	tmNode *tmNode.Node

	// currentMempool is responsive for prevent sending multiple transactions from one address in one block
	currentMempool *sync.Map

	lock sync.RWMutex

	haltHeight uint64
	cfg        *config.Config
	storages   *utils.Storage
}

// NewMinterBlockchain creates Minter Blockchain instance, should be only called once
func NewMinterBlockchain(storages *utils.Storage, cfg *config.Config) *Blockchain {
	// Initiate Application DB. Used for persisting data like current block, validators, etc.
	applicationDB := appdb.NewAppDB(storages.GetMinterHome(), cfg)

	lastHeight := applicationDB.GetLastHeight()
	initialHeight := applicationDB.GetStartHeight()
	height := lastHeight
	if lastHeight < initialHeight {
		height = initialHeight
	}
	blockchain := &Blockchain{
		appDB:          applicationDB,
		storages:       storages,
		height:         height,
		eventsDB:       eventsdb.NewEventsStore(storages.EventDB()),
		currentMempool: &sync.Map{},
		cfg:            cfg,
	}

	var err error
	// Set stateDeliver and stateCheck
	blockchain.stateDeliver, err = state.NewState(lastHeight, blockchain.storages.StateDB(), blockchain.eventsDB, cfg.StateCacheSize, cfg.KeepLastStates, initialHeight)
	if err != nil {
		panic(err)
	}

	blockchain.stateCheck = state.NewCheckState(blockchain.stateDeliver)

	// Set start height for rewards and validators
	rewards.SetStartHeight(initialHeight)

	blockchain.haltHeight = uint64(cfg.HaltHeight)

	return blockchain
}

// InitChain initialize blockchain with validators and other info. Only called once.
func (blockchain *Blockchain) InitChain(req abciTypes.RequestInitChain) abciTypes.ResponseInitChain {
	var genesisState types.AppState
	if err := amino.UnmarshalJSON(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	if genesisState.StartHeight > blockchain.height {
		blockchain.appDB.SetStartHeight(genesisState.StartHeight)
		blockchain.Close()
		*blockchain = *NewMinterBlockchain(blockchain.storages, blockchain.cfg)
	}
	if err := blockchain.stateDeliver.Import(genesisState); err != nil {
		panic(err)
	}
	_, err := blockchain.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	vals := blockchain.updateValidators()

	blockchain.appDB.SetStartHeight(genesisState.StartHeight)
	blockchain.appDB.FlushValidators()

	rewards.SetStartHeight(genesisState.StartHeight)

	return abciTypes.ResponseInitChain{
		Validators: vals,
	}
}

// BeginBlock signals the beginning of a block.
func (blockchain *Blockchain) BeginBlock(req abciTypes.RequestBeginBlock) abciTypes.ResponseBeginBlock {

	height := uint64(req.Header.Height)

	blockchain.StatisticData().PushStartBlock(&statistics.StartRequest{Height: int64(height), Now: time.Now(), HeaderTime: req.Header.Time})
	blockchain.stateDeliver.Lock()

	// compute max gas
	blockchain.updateBlocksTimeDelta(height, 3)
	maxGas := blockchain.calcMaxGas(height)
	blockchain.stateDeliver.App.SetMaxGas(maxGas)

	atomic.StoreUint64(&blockchain.height, height)
	blockchain.rewards = big.NewInt(0)

	// clear absent candidates
	blockchain.lock.Lock()
	blockchain.validatorsStatuses = map[types.TmAddress]int8{}

	// give penalty to absent validators
	for _, v := range req.LastCommitInfo.Votes {
		var address types.TmAddress
		copy(address[:], v.Validator.Address)

		if v.SignedLastBlock {
			blockchain.stateDeliver.Validators.SetValidatorPresent(height, address)
			blockchain.validatorsStatuses[address] = ValidatorPresent
		} else {
			blockchain.stateDeliver.Validators.SetValidatorAbsent(height, address)
			blockchain.validatorsStatuses[address] = ValidatorAbsent
		}
	}
	blockchain.lock.Unlock()

	if blockchain.isApplicationHalted(height) {
		panic(fmt.Sprintf("Application halted at height %d", height))
	}

	// give penalty to Byzantine validators
	for _, byzVal := range req.ByzantineValidators {
		var address types.TmAddress
		copy(address[:], byzVal.Validator.Address)

		// skip already offline candidates to prevent double punishing
		candidate := blockchain.stateDeliver.Candidates.GetCandidateByTendermintAddress(address)
		if candidate == nil || candidate.Status == candidates.CandidateStatusOffline || blockchain.stateDeliver.Validators.GetByTmAddress(address) == nil {
			continue
		}

		blockchain.stateDeliver.FrozenFunds.PunishFrozenFundsWithID(height, height+types.GetUnbondPeriod(), candidate.ID)
		blockchain.stateDeliver.Validators.PunishByzantineValidator(address)
		blockchain.stateDeliver.Candidates.PunishByzantineCandidate(height, address)
	}

	// apply frozen funds (used for unbond stakes)
	frozenFunds := blockchain.stateDeliver.FrozenFunds.GetFrozenFunds(uint64(req.Header.Height))
	if frozenFunds != nil {
		for _, item := range frozenFunds.List {
			amount := item.Value
			if item.MoveToCandidate == nil {
				blockchain.eventsDB.AddEvent(uint32(req.Header.Height), &eventsdb.UnbondEvent{
					Address:         item.Address,
					Amount:          amount.String(),
					Coin:            uint64(item.Coin),
					ValidatorPubKey: *item.CandidateKey,
				})
				blockchain.stateDeliver.Accounts.AddBalance(item.Address, item.Coin, amount)
			} else {
				newCandidate := blockchain.stateDeliver.Candidates.PubKey(*item.MoveToCandidate)
				value := big.NewInt(0).Set(amount)
				if wl := blockchain.stateDeliver.Waitlist.Get(item.Address, newCandidate, item.Coin); wl != nil {
					value.Add(value, wl.Value)
					blockchain.stateDeliver.Waitlist.Delete(item.Address, newCandidate, item.Coin)
				}
				var toWaitlist bool
				if blockchain.stateDeliver.Candidates.IsDelegatorStakeSufficient(item.Address, newCandidate, item.Coin, value) {
					blockchain.stateDeliver.Candidates.Delegate(item.Address, newCandidate, item.Coin, value, big.NewInt(0))
				} else {
					blockchain.stateDeliver.Waitlist.AddWaitList(item.Address, newCandidate, item.Coin, value)
					toWaitlist = true
				}
				blockchain.eventsDB.AddEvent(uint32(req.Header.Height), &eventsdb.StakeMoveEvent{
					Address:         item.Address,
					Amount:          amount.String(),
					Coin:            uint64(item.Coin),
					ValidatorPubKey: *item.CandidateKey,
					WaitList:        toWaitlist,
				})
			}
		}

		// delete from db
		blockchain.stateDeliver.FrozenFunds.Delete(frozenFunds.Height())
	}

	blockchain.stateDeliver.Halts.Delete(height)

	return abciTypes.ResponseBeginBlock{}
}

// EndBlock signals the end of a block, returns changes to the validator set
func (blockchain *Blockchain) EndBlock(req abciTypes.RequestEndBlock) abciTypes.ResponseEndBlock {
	height := uint64(req.Height)

	vals := blockchain.stateDeliver.Validators.GetValidators()

	hasDroppedValidators := false
	for _, val := range vals {
		if !val.IsToDrop() {
			continue
		}
		hasDroppedValidators = true

		// Move dropped validator's accum rewards back to pool
		blockchain.rewards.Add(blockchain.rewards, val.GetAccumReward())
		val.SetAccumReward(big.NewInt(0))
	}

	// calculate total power of validators
	totalPower := big.NewInt(0)
	for _, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || blockchain.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
			continue
		}

		totalPower.Add(totalPower, val.GetTotalBipStake())
	}

	if totalPower.Sign() == 0 {
		totalPower = big.NewInt(1)
	}

	// accumulate rewards
	reward := rewards.GetRewardForBlock(height)
	blockchain.stateDeliver.Checker.AddCoinVolume(types.GetBaseCoinID(), reward)
	reward.Add(reward, blockchain.rewards)

	// compute remainder to keep total emission consist
	remainder := big.NewInt(0).Set(reward)

	for i, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || blockchain.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
			continue
		}

		r := big.NewInt(0).Set(reward)
		r.Mul(r, val.GetTotalBipStake())
		r.Div(r, totalPower)

		remainder.Sub(remainder, r)
		vals[i].AddAccumReward(r)
	}

	// add remainder to total slashed
	blockchain.stateDeliver.App.AddTotalSlashed(remainder)

	// pay rewards
	if height%120 == 0 {
		blockchain.stateDeliver.Validators.PayRewards(height)
	}

	hasChangedPublicKeys := false
	if blockchain.stateDeliver.Candidates.IsChangedPublicKeys() {
		blockchain.stateDeliver.Candidates.ResetIsChangedPublicKeys()
		hasChangedPublicKeys = true
	}

	// update validators
	var updates []abciTypes.ValidatorUpdate
	if height%120 == 0 || hasDroppedValidators || hasChangedPublicKeys {
		updates = blockchain.updateValidators()
	}

	defer func() {
		blockchain.StatisticData().PushEndBlock(&statistics.EndRequest{TimeEnd: time.Now(), Height: int64(blockchain.Height())})
	}()

	return abciTypes.ResponseEndBlock{
		ValidatorUpdates: updates,
		ConsensusParamUpdates: &abciTypes.ConsensusParams{
			Block: &abciTypes.BlockParams{
				MaxBytes: blockMaxBytes,
				MaxGas:   int64(blockchain.stateDeliver.App.GetMaxGas()),
			},
		},
	}
}

func (blockchain *Blockchain) updateValidators() []abciTypes.ValidatorUpdate {
	height := blockchain.Height()
	blockchain.stateDeliver.Candidates.RecalculateStakes(height)

	valsCount := validators.GetValidatorsCountForBlock(height)
	newCandidates := blockchain.stateDeliver.Candidates.GetNewCandidates(valsCount)
	if len(newCandidates) < valsCount {
		valsCount = len(newCandidates)
	}

	newValidators := make([]abciTypes.ValidatorUpdate, 0, valsCount)

	// calculate total power
	totalPower := big.NewInt(0)
	for _, candidate := range newCandidates {
		totalPower.Add(totalPower, blockchain.stateDeliver.Candidates.GetTotalStake(candidate.PubKey))
	}

	for _, newCandidate := range newCandidates {
		power := big.NewInt(0).Div(big.NewInt(0).Mul(blockchain.stateDeliver.Candidates.GetTotalStake(newCandidate.PubKey),
			big.NewInt(100000000)), totalPower).Int64()

		if power == 0 {
			power = 1
		}

		newValidators = append(newValidators, abciTypes.Ed25519ValidatorUpdate(newCandidate.PubKey.Bytes(), power))
	}

	sort.SliceStable(newValidators, func(i, j int) bool {
		return newValidators[i].Power > newValidators[j].Power
	})

	// update validators in state
	blockchain.stateDeliver.Validators.SetNewValidators(newCandidates)

	activeValidators := blockchain.appDB.GetValidators()

	blockchain.appDB.SetValidators(newValidators)

	updates := newValidators

	for _, validator := range activeValidators {
		persisted := false
		for _, newValidator := range newValidators {
			if bytes.Equal(validator.PubKey.Data, newValidator.PubKey.Data) {
				persisted = true
				break
			}
		}

		// remove validator
		if !persisted {
			updates = append(updates, abciTypes.ValidatorUpdate{
				PubKey: validator.PubKey,
				Power:  0,
			})
		}
	}
	return updates
}

// Info return application info. Used for synchronization between Tendermint and Minter
func (blockchain *Blockchain) Info(_ abciTypes.RequestInfo) (resInfo abciTypes.ResponseInfo) {
	return abciTypes.ResponseInfo{
		Version:          version.Version,
		AppVersion:       version.AppVer,
		LastBlockHeight:  int64(blockchain.appDB.GetLastHeight()),
		LastBlockAppHash: blockchain.appDB.GetLastBlockHash(),
	}
}

// DeliverTx deliver a tx for full processing
func (blockchain *Blockchain) DeliverTx(req abciTypes.RequestDeliverTx) abciTypes.ResponseDeliverTx {
	response := transaction.RunTx(blockchain.stateDeliver, req.Tx, blockchain.rewards, blockchain.Height(), &sync.Map{}, 0)

	return abciTypes.ResponseDeliverTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Events: []abciTypes.Event{
			{
				Type:       "tags",
				Attributes: response.Tags,
			},
		},
	}
}

// CheckTx validates a tx for the mempool
func (blockchain *Blockchain) CheckTx(req abciTypes.RequestCheckTx) abciTypes.ResponseCheckTx {
	response := transaction.RunTx(blockchain.CurrentState(), req.Tx, nil, blockchain.height, blockchain.currentMempool, blockchain.MinGasPrice())

	return abciTypes.ResponseCheckTx{
		Code:      response.Code,
		Data:      response.Data,
		Log:       response.Log,
		Info:      response.Info,
		GasWanted: response.GasWanted,
		GasUsed:   response.GasUsed,
		Events: []abciTypes.Event{
			{
				Type:       "tags",
				Attributes: response.Tags,
			},
		},
	}
}

// Commit the state and return the application Merkle root hash
func (blockchain *Blockchain) Commit() abciTypes.ResponseCommit {
	if err := blockchain.stateDeliver.Check(); err != nil {
		panic(err)
	}

	// Committing Minter Blockchain state
	hash, err := blockchain.stateDeliver.Commit()
	if err != nil {
		panic(err)
	}

	// Flush events db
	err = blockchain.eventsDB.CommitEvents()
	if err != nil {
		panic(err)
	}

	// Persist application hash and height
	blockchain.appDB.SetLastBlockHash(hash)
	blockchain.appDB.SetLastHeight(blockchain.Height())
	blockchain.appDB.FlushValidators()

	blockchain.stateDeliver.Unlock()

	// Resetting check state to be consistent with current height
	blockchain.resetCheckState()

	// Clear mempool
	blockchain.currentMempool = &sync.Map{}

	return abciTypes.ResponseCommit{
		Data: hash,
	}
}

// Query Unused method, required by Tendermint
func (blockchain *Blockchain) Query(_ abciTypes.RequestQuery) abciTypes.ResponseQuery {
	return abciTypes.ResponseQuery{}
}

// SetOption Unused method, required by Tendermint
func (blockchain *Blockchain) SetOption(_ abciTypes.RequestSetOption) abciTypes.ResponseSetOption {
	return abciTypes.ResponseSetOption{}
}

// Stop gracefully stopping Minter Blockchain instance
func (blockchain *Blockchain) Stop() {
	err := blockchain.tmNode.Stop()
	if err != nil {
		panic(err)
	}
	blockchain.Close()
}

// Close closes db connections
func (blockchain *Blockchain) Close() {
	blockchain.appDB.Close()
	if err := blockchain.storages.StateDB().Close(); err != nil {
		panic(err)
	}
	if err := blockchain.storages.EventDB().Close(); err != nil {
		panic(err)
	}
}

// CurrentState returns immutable state of Minter Blockchain
func (blockchain *Blockchain) CurrentState() *state.CheckState {
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()

	return blockchain.stateCheck
}

// AvailableVersions returns all available versions in ascending order
func (blockchain *Blockchain) AvailableVersions() []int {
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()

	return blockchain.stateDeliver.Tree().AvailableVersions()
}

// GetStateForHeight returns immutable state of Minter Blockchain for given height
func (blockchain *Blockchain) GetStateForHeight(height uint64) (*state.CheckState, error) {
	if height > 0 {
		s, err := state.NewCheckStateAtHeight(height, blockchain.storages.StateDB())
		if err != nil {
			return nil, err
		}
		return s, nil
	}
	return blockchain.CurrentState(), nil
}

// Height returns current height of Minter Blockchain
func (blockchain *Blockchain) Height() uint64 {
	return atomic.LoadUint64(&blockchain.height)
}

// SetTmNode sets Tendermint node
func (blockchain *Blockchain) SetTmNode(node *tmNode.Node) {
	blockchain.tmNode = node
}

// MinGasPrice returns minimal acceptable gas price
func (blockchain *Blockchain) MinGasPrice() uint32 {
	mempoolSize := blockchain.tmNode.Mempool().Size()

	if mempoolSize > 5000 {
		return 50
	}

	if mempoolSize > 1000 {
		return 10
	}

	if mempoolSize > 500 {
		return 5
	}

	if mempoolSize > 100 {
		return 2
	}

	return 1
}

func (blockchain *Blockchain) resetCheckState() {
	blockchain.lock.Lock()
	defer blockchain.lock.Unlock()

	blockchain.stateCheck = state.NewCheckState(blockchain.stateDeliver)
}

func (blockchain *Blockchain) updateBlocksTimeDelta(height uint64, count int64) {
	// should do this because tmNode is unavailable during Tendermint's replay mode
	if blockchain.tmNode == nil {
		return
	}

	if int64(height)-count-1 < 1 {
		return
	}

	blockStore := blockchain.tmNode.BlockStore()

	blockA := blockStore.LoadBlockMeta(int64(height) - count - 1)
	blockB := blockStore.LoadBlockMeta(int64(height) - 1)

	delta := int(blockB.Header.Time.Sub(blockA.Header.Time).Seconds())
	blockchain.appDB.SetLastBlocksTimeDelta(height, delta)
}

// SetBlocksTimeDelta sets current blocks time delta
func (blockchain *Blockchain) SetBlocksTimeDelta(height uint64, value int) {
	blockchain.appDB.SetLastBlocksTimeDelta(height, value)
}

// GetBlocksTimeDelta returns current blocks time delta
func (blockchain *Blockchain) GetBlocksTimeDelta(height, _ uint64) (int, error) {
	return blockchain.appDB.GetLastBlocksTimeDelta(height)
}

func (blockchain *Blockchain) calcMaxGas(height uint64) uint64 {
	const targetTime = 7
	const blockDelta = 3

	// skip first 20 blocks
	if height <= 20 {
		return defaultMaxGas
	}

	// get current max gas
	newMaxGas := blockchain.stateCheck.App().GetMaxGas()

	// check if blocks are created in time
	if delta, _ := blockchain.GetBlocksTimeDelta(height, blockDelta); delta > targetTime*blockDelta {
		newMaxGas = newMaxGas * 7 / 10 // decrease by 30%
	} else {
		newMaxGas = newMaxGas * 105 / 100 // increase by 5%
	}

	// check if max gas is too high
	if newMaxGas > defaultMaxGas {
		newMaxGas = defaultMaxGas
	}

	// check if max gas is too low
	if newMaxGas < minMaxGas {
		newMaxGas = minMaxGas
	}

	return newMaxGas
}

// GetEventsDB returns current EventsDB
func (blockchain *Blockchain) GetEventsDB() eventsdb.IEventsDB {
	return blockchain.eventsDB
}

// SetStatisticData used for collection statistics about blockchain operations
func (blockchain *Blockchain) SetStatisticData(statisticData *statistics.Data) *statistics.Data {
	blockchain.statisticData = statisticData
	return blockchain.statisticData
}

// StatisticData used for collection statistics about blockchain operations
func (blockchain *Blockchain) StatisticData() *statistics.Data {
	return blockchain.statisticData
}

// GetValidatorStatus returns given validator's status
func (blockchain *Blockchain) GetValidatorStatus(address types.TmAddress) int8 {
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()

	return blockchain.validatorsStatuses[address]
}

// DeleteStateVersions deletes states in given range
func (blockchain *Blockchain) DeleteStateVersions(from, to int64) error {
	blockchain.lock.RLock()
	defer blockchain.lock.RUnlock()

	return blockchain.stateDeliver.Tree().DeleteVersionsRange(from, to)
}

func (blockchain *Blockchain) isApplicationHalted(height uint64) bool {
	if blockchain.haltHeight > 0 && height >= blockchain.haltHeight {
		return true
	}

	halts := blockchain.stateDeliver.Halts.GetHaltBlocks(height)
	if halts != nil {
		// calculate total power of validators
		vals := blockchain.stateDeliver.Validators.GetValidators()
		totalPower, totalVotedPower := big.NewInt(0), big.NewInt(0)
		for _, val := range vals {
			// skip if candidate is not present
			if val.IsToDrop() || blockchain.validatorsStatuses[val.GetAddress()] != ValidatorPresent {
				continue
			}

			for _, halt := range halts.List {
				if halt.Pubkey == val.PubKey {
					totalVotedPower.Add(totalVotedPower, val.GetTotalBipStake())
				}
			}

			totalPower.Add(totalPower, val.GetTotalBipStake())
		}

		if totalPower.Cmp(types.Big0) == 0 {
			totalPower = big.NewInt(1)
		}

		votingResult := new(big.Float).Quo(
			new(big.Float).SetInt(totalVotedPower),
			new(big.Float).SetInt(totalPower),
		)

		if votingResult.Cmp(big.NewFloat(votingPowerConsensus)) == 1 {
			return true
		}
	}

	return false
}

func GetDbOpts(memLimit int) *opt.Options {
	if memLimit < 1024 {
		panic(fmt.Sprintf("Not enough memory given to StateDB. Expected >1024M, given %d", memLimit))
	}
	return &opt.Options{
		OpenFilesCacheCapacity: memLimit,
		BlockCacheCapacity:     memLimit / 2 * opt.MiB,
		WriteBuffer:            memLimit / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	}
}
