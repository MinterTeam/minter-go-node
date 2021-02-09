package minter

import (
	"fmt"
	eventsdb "github.com/MinterTeam/minter-go-node/core/events"
	"github.com/MinterTeam/minter-go-node/core/state"
	validators2 "github.com/MinterTeam/minter-go-node/core/state/validators"
	"github.com/MinterTeam/minter-go-node/core/statistics"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/validators"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	abciTypes "github.com/tendermint/tendermint/abci/types"
	tmNode "github.com/tendermint/tendermint/node"
	"log"
	"math/big"
	"sort"
	"sync/atomic"
)

func (blockchain *Blockchain) checkStop() bool {
	if !blockchain.stopped {
		select {
		case <-blockchain.stopChan.Done():
			blockchain.stop()
		default:
		}
	}
	return blockchain.stopped
}

func (blockchain *Blockchain) stop() {
	blockchain.stopped = true
	go func() {
		log.Println("Stopping Node")
		log.Println("Node Stopped with error:", blockchain.tmNode.Stop())
	}()
}

// Stop gracefully stopping Minter Blockchain instance
func (blockchain *Blockchain) WaitStop() error {
	blockchain.tmNode.Wait()
	return blockchain.Close()
}

// calculatePowers calculates total power of validators
func (blockchain *Blockchain) calculatePowers(vals []*validators2.Validator) {
	blockchain.validatorsPowers = map[types.Pubkey]*big.Int{}
	blockchain.totalPower = big.NewInt(0)
	for _, val := range vals {
		// skip if candidate is not present
		if val.IsToDrop() || blockchain.GetValidatorStatus(val.GetAddress()) != ValidatorPresent {
			continue
		}

		blockchain.validatorsPowers[val.PubKey] = val.GetTotalBipStake()
		blockchain.totalPower.Add(blockchain.totalPower, val.GetTotalBipStake())
	}

	if blockchain.totalPower.Sign() == 0 {
		blockchain.totalPower = big.NewInt(1)
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
			if validator.PubKey.Sum.Compare(newValidator.PubKey.Sum) == 0 {
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

func (blockchain *Blockchain) updateBlocksTimeDelta(height uint64) {
	// should do this because tmNode is unavailable during Tendermint's replay mode
	if blockchain.tmNode == nil {
		return
	}

	blockStore := blockchain.tmNode.BlockStore()
	baseMeta := blockStore.LoadBaseMeta()
	if int64(height)-1 < baseMeta.Header.Height {
		return
	}

	blockA := blockStore.LoadBlockMeta(int64(height) - 1)
	blockB := blockStore.LoadBlockMeta(int64(height))

	delta := int(blockB.Header.Time.Sub(blockA.Header.Time).Seconds())
	blockchain.appDB.AddBlocksTimeDelta(height, delta)
}

func (blockchain *Blockchain) calcMaxGas(height uint64) uint64 {
	const targetTime = 7
	const blockDelta = 3

	// get current max gas
	newMaxGas := blockchain.stateCheck.App().GetMaxGas()

	// check if blocks are created in time
	delta, err := blockchain.appDB.GetLastBlocksTimeDelta(height)
	if err != nil {
		log.Println(err)
		return defaultMaxGas
	}

	if delta > targetTime*blockDelta {
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
	if halts == nil {
		return false
	}

	totalVotedPower := big.NewInt(0)
	for _, halt := range halts.List {
		if power, ok := blockchain.validatorsPowers[halt.Pubkey]; ok {
			totalVotedPower.Add(totalVotedPower, power)
		}
	}

	votingResult := new(big.Float).Quo(
		new(big.Float).SetInt(totalVotedPower),
		new(big.Float).SetInt(blockchain.totalPower),
	)

	if votingResult.Cmp(big.NewFloat(votingPowerConsensus)) == 1 {
		return true
	}

	return false
}

func (blockchain *Blockchain) isUpdateCommissionsBlock(height uint64) []byte {
	commissions := blockchain.stateDeliver.Commission.GetVotes(height)
	if len(commissions) == 0 {
		return nil
	}
	// calculate total power of validators
	maxVotingResult, totalVotedPower := big.NewFloat(0), big.NewInt(0)

	var price string
	for _, commission := range commissions {
		for _, vote := range commission.Votes {
			if power, ok := blockchain.validatorsPowers[vote]; ok {
				totalVotedPower.Add(totalVotedPower, power)
			}
		}
		votingResult := new(big.Float).Quo(
			new(big.Float).SetInt(totalVotedPower),
			new(big.Float).SetInt(blockchain.totalPower),
		)

		if maxVotingResult.Cmp(votingResult) == -1 {
			maxVotingResult = votingResult
			price = commission.Price
		}
	}
	if maxVotingResult.Cmp(big.NewFloat(votingPowerConsensus)) == 1 {
		return []byte(price)
	}

	return nil
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
