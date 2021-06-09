package mempool

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	tmpool "github.com/tendermint/tendermint/mempool"
	"sort"
	"sync"
	"sync/atomic"

	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	auto "github.com/tendermint/tendermint/libs/autofile"
	"github.com/tendermint/tendermint/libs/clist"
	"github.com/tendermint/tendermint/libs/log"
	tmmath "github.com/tendermint/tendermint/libs/math"
	tmos "github.com/tendermint/tendermint/libs/os"
	tmsync "github.com/tendermint/tendermint/libs/sync"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"
)

// TxKeySize is the size of the transaction key index
const TxKeySize = sha256.Size

var newline = []byte("\n")

//--------------------------------------------------------------------------------

// PriorityMempool is an ordered in-memory pool for transactions before they are
// proposed in a consensus round. Transaction validity is checked using the
// CheckTx abci message before the transaction is added to the pool. The
// mempool uses a concurrent list structure for storing transactions that can
// be efficiently accessed by multiple concurrent readers.
type PriorityMempool struct {
	// Atomic integers
	height   int64 // the last block Update()'d to
	txsBytes int64 // total size of mempool, in bytes

	// notify listeners (ie. consensus) when txs are available
	notifiedTxsAvailable bool
	txsAvailable         chan struct{} // fires once for each height, when the mempool is not empty

	config *cfg.MempoolConfig

	// Exclusive mutex for Update method to prevent concurrent execution of
	// CheckTx or ReapMaxBytesMaxGas(ReapMaxTxs) methods.
	updateMtx tmsync.RWMutex
	preCheck  tmpool.PreCheckFunc
	postCheck tmpool.PostCheckFunc

	wal                *auto.AutoFile          // a log of mempool txs
	txs                map[uint32]*clist.CList // concurrent linked-list of good txs
	txsGasPriceCounter map[uint32]uint64
	gasPrices          []uint32
	txsCounter         int
	proxyAppConn       proxy.AppConnMempool

	// locks
	txsmx  sync.RWMutex
	txscmx sync.RWMutex

	// Track whether we're rechecking txs.
	// These are not protected by a mutex and are expected to be mutated in
	// serial (ie. by abci responses which are called in serial).
	recheckCursor *clist.CElement // next expected response
	recheckEnd    *clist.CElement // re-checking stops here

	// Map for quick access to txs to record sender in CheckTx.
	// txsMap: txKey -> CElement
	txsMap sync.Map

	// Keep a cache of already-seen txs.
	// This reduces the pressure on the proxyApp.
	cache txCache

	logger log.Logger

	metrics  *tmpool.Metrics
	executor *transaction.Executor
}

// PriorityMempoolOption sets an optional parameter on the tmpool.
type PriorityMempoolOption func(*PriorityMempool)

// NewPriorityMempool returns a new mempool with the given configuration and connection to an application.
func NewPriorityMempool(
	config *cfg.MempoolConfig,
	proxyAppConn proxy.AppConnMempool,
	height int64,
	options ...PriorityMempoolOption,
) *PriorityMempool {
	mempool := &PriorityMempool{
		config:             config,
		proxyAppConn:       proxyAppConn,
		txs:                make(map[uint32]*clist.CList),
		txsGasPriceCounter: make(map[uint32]uint64),
		txsCounter:         0,
		height:             height,
		recheckCursor:      nil,
		recheckEnd:         nil,
		logger:             log.NewNopLogger(),
		metrics:            tmpool.NopMetrics(),
		executor:           transaction.NewExecutor(transaction.GetData),
	}
	if config.CacheSize > 0 {
		mempool.cache = newMapTxCache(config.CacheSize)
	} else {
		mempool.cache = nopTxCache{}
	}
	proxyAppConn.SetResponseCallback(mempool.globalCb)
	for _, option := range options {
		option(mempool)
	}
	return mempool
}

// NOTE: not thread safe - should only be called once, on startup
func (mem *PriorityMempool) EnableTxsAvailable() {
	mem.txsAvailable = make(chan struct{}, 1)
}

// SetLogger sets the Logger.
func (mem *PriorityMempool) SetLogger(l log.Logger) {
	mem.logger = l
}

// WithPreCheck sets a filter for the mempool to reject a tx if f(tx) returns
// false. This is ran before CheckTx. Only applies to the first created block.
// After that, Update overwrites the existing value.
func WithPreCheck(f tmpool.PreCheckFunc) PriorityMempoolOption {
	return func(mem *PriorityMempool) { mem.preCheck = f }
}

// WithPostCheck sets a filter for the mempool to reject a tx if f(tx) returns
// false. This is ran after CheckTx. Only applies to the first created block.
// After that, Update overwrites the existing value.
func WithPostCheck(f tmpool.PostCheckFunc) PriorityMempoolOption {
	return func(mem *PriorityMempool) { mem.postCheck = f }
}

// WithMetrics sets the metrics.
func WithMetrics(metrics *tmpool.Metrics) PriorityMempoolOption {
	return func(mem *PriorityMempool) { mem.metrics = metrics }
}

func (mem *PriorityMempool) InitWAL() error {
	var (
		walDir  = mem.config.WalDir()
		walFile = walDir + "/wal"
	)

	const perm = 0700
	if err := tmos.EnsureDir(walDir, perm); err != nil {
		return err
	}

	af, err := auto.OpenAutoFile(walFile)
	if err != nil {
		return fmt.Errorf("can't open autofile %s: %w", walFile, err)
	}

	mem.wal = af
	return nil
}

func (mem *PriorityMempool) CloseWAL() {
	if err := mem.wal.Close(); err != nil {
		mem.logger.Error("Error closing WAL", "err", err)
	}
	mem.wal = nil
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) Lock() {
	mem.updateMtx.Lock()
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) Unlock() {
	mem.updateMtx.Unlock()
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) Size() int {
	return mem.txsCounter
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) TxsBytes() int64 {
	return atomic.LoadInt64(&mem.txsBytes)
}

func (mem *PriorityMempool) txsByGasPrices() ([]uint32, uint64) {
	gasPrices, txCount := make([]uint32, 0), uint64(0)
	for gasPrice, count := range mem.txsGasPriceCounter {
		if count > 0 {
			txCount += count
			gasPrices = append(gasPrices, gasPrice)
		}
	}

	sort.Slice(gasPrices, func(i, j int) bool { return gasPrices[i] > gasPrices[j] })
	return gasPrices, txCount
}

func (mem *PriorityMempool) incrementTxsCounter(gasPrice uint32) {
	mem.txscmx.Lock()
	defer mem.txscmx.Unlock()

	mem.txsCounter += 1
	if _, ok := mem.txsGasPriceCounter[gasPrice]; ok {
		mem.txsGasPriceCounter[gasPrice] += 1
		return
	}

	mem.txsGasPriceCounter[gasPrice] = 1
}

func (mem *PriorityMempool) decrementTxsCounter(gasPrice uint32) {
	mem.txscmx.Lock()
	defer mem.txscmx.Unlock()

	mem.txsCounter -= 1
	mem.txsGasPriceCounter[gasPrice] -= 1
}

// Lock() must be help by the caller during execution.
func (mem *PriorityMempool) FlushAppConn() error {
	return mem.proxyAppConn.FlushSync()
}

// XXX: Unsafe! Calling Flush may leave mempool in inconsistent state.
func (mem *PriorityMempool) Flush() {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	_ = atomic.SwapInt64(&mem.txsBytes, 0)
	mem.cache.Reset()

	gasPrices, _ := mem.txsByGasPrices()
	for _, gp := range gasPrices {
		for e := mem.getTxByGasPrice(gp).Front(); e != nil; e = e.Next() {
			mem.txsmx.Lock()
			mem.txs[gp].Remove(e)
			mem.txsmx.Unlock()

			e.DetachPrev()
			mem.decrementTxsCounter(gp)
		}
	}

	mem.txsMap.Range(func(key, _ interface{}) bool {
		mem.txsMap.Delete(key)
		return true
	})
}

// It blocks if we're waiting on Update() or Reap().
// cb: A callback from the CheckTx command.
//     It gets called from another goroutine.
// CONTRACT: Either cb will get called, or err returned.
//
// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) CheckTx(tx types.Tx, cb func(*abci.Response), txInfo tmpool.TxInfo) error {
	mem.updateMtx.RLock()
	// use defer to unlock mutex because application (*local client*) might panic
	defer mem.updateMtx.RUnlock()

	txSize := len(tx)

	if err := mem.isFull(txSize); err != nil {
		return err
	}

	if txSize > mem.config.MaxTxBytes {
		return ErrTxTooLarge{mem.config.MaxTxBytes, txSize}
	}

	if mem.preCheck != nil {
		if err := mem.preCheck(tx); err != nil {
			return tmpool.ErrPreCheck{err}
		}
	}

	// NOTE: writing to the WAL and calling proxy must be done before adding tx
	// to the cache. otherwise, if either of them fails, next time CheckTx is
	// called with tx, ErrTxInCache will be returned without tx being checked at
	// all even once.
	if mem.wal != nil {
		// TODO: Notify administrators when WAL fails
		_, err := mem.wal.Write(append([]byte(tx), newline...))
		if err != nil {
			return fmt.Errorf("wal.Write: %w", err)
		}
	}

	// NOTE: proxyAppConn may error if tx buffer is full
	if err := mem.proxyAppConn.Error(); err != nil {
		return err
	}

	if !mem.cache.Push(tx) {
		// Record a new sender for a tx we've already seen.
		// Note it's possible a tx is still in the cache but no longer in the mempool
		// (eg. after committing a block, txs are removed from mempool but not cache),
		// so we only record the sender for txs still in the tmpool.
		if e, ok := mem.txsMap.Load(TxKey(tx)); ok {
			memTx := e.(*clist.CElement).Value.(*mempoolTx)
			memTx.senders.LoadOrStore(txInfo.SenderID, true)
			// TODO: consider punishing peer for dups,
			// its non-trivial since invalid txs can become valid,
			// but they can spam the same tx with little cost to them atm.
		}

		return tmpool.ErrTxInCache
	}

	reqRes := mem.proxyAppConn.CheckTxAsync(abci.RequestCheckTx{Tx: tx})
	reqRes.SetCallback(mem.reqResCb(tx, txInfo.SenderID, txInfo.SenderP2PID, cb))

	return nil
}

// Global callback that will be called after every ABCI response.
// Having a single global callback avoids needing to set a callback for each request.
// However, processing the checkTx response requires the peerID (so we can track which txs we heard from who),
// and peerID is not included in the ABCI request, so we have to set request-specific callbacks that
// include this information. If we're not in the midst of a recheck, this function will just return,
// so the request specific callback can do the work.
//
// When rechecking, we don't need the peerID, so the recheck callback happens
// here.
func (mem *PriorityMempool) globalCb(req *abci.Request, res *abci.Response) {
	if mem.recheckCursor == nil {
		return
	}

	mem.metrics.RecheckTimes.Add(1)
	mem.resCbRecheck(req, res)

	// update metrics
	mem.metrics.Size.Set(float64(mem.Size()))
}

// Request specific callback that should be set on individual reqRes objects
// to incorporate local information when processing the response.
// This allows us to track the peer that sent us this tx, so we can avoid sending it back to them.
// NOTE: alternatively, we could include this information in the ABCI request itself.
//
// External callers of CheckTx, like the RPC, can also pass an externalCb through here that is called
// when all other response processing is complete.
//
// Used in CheckTx to record PeerID who sent us the tx.
func (mem *PriorityMempool) reqResCb(
	tx []byte,
	peerID uint16,
	peerP2PID p2p.ID,
	externalCb func(*abci.Response),
) func(res *abci.Response) {
	return func(res *abci.Response) {
		if mem.recheckCursor != nil {
			// this should never happen
			panic("recheck cursor is not nil in reqResCb")
		}

		mem.resCbFirstTime(tx, peerID, peerP2PID, res)

		// update metrics
		mem.metrics.Size.Set(float64(mem.Size()))

		// passed in by the caller of CheckTx, eg. the RPC
		if externalCb != nil {
			externalCb(res)
		}
	}
}

func (mem *PriorityMempool) getTxByGasPrice(gp uint32) *clist.CList {
	mem.txsmx.RLock()
	defer mem.txsmx.RUnlock()
	return mem.txs[gp]
}

func (mem *PriorityMempool) addGasPrice(gp uint32) {
	exists := false
	i := sort.Search(len(mem.gasPrices), func(i int) bool {
		if mem.gasPrices[i] == gp {
			exists = true
		}
		return mem.gasPrices[i] > gp
	})

	if exists {
		return
	}

	mem.gasPrices = append(mem.gasPrices, 0)
	copy(mem.gasPrices[i+1:], mem.gasPrices[i:])
	mem.gasPrices[i] = gp
}

// Called from:
//  - resCbFirstTime (lock not held) if tx is valid
func (mem *PriorityMempool) addTx(memTx *mempoolTx) {
	tx, err := mem.executor.DecodeFromBytes(memTx.tx) // TODO: handle error
	if err != nil {
		panic(fmt.Sprintf("failed to decode tx: %X", memTx.tx))
	}

	mem.txsmx.Lock()
	if _, ok := mem.txs[tx.GasPrice]; !ok {
		mem.txs[tx.GasPrice] = clist.New()
	}
	e := mem.txs[tx.GasPrice].PushBack(memTx)
	mem.txsmx.Unlock()

	mem.incrementTxsCounter(tx.GasPrice)
	mem.addGasPrice(tx.GasPrice)

	mem.txsMap.Store(TxKey(memTx.tx), e)
	atomic.AddInt64(&mem.txsBytes, int64(len(memTx.tx)))
	mem.metrics.TxSizeBytes.Observe(float64(len(memTx.tx)))
}

// Called from:
//  - Update (lock held) if tx was committed
// 	- resCbRecheck (lock not held) if tx was invalidated
func (mem *PriorityMempool) removeTx(tx types.Tx, elem *clist.CElement, removeFromCache bool) {
	decodedTx, _ := mem.executor.DecodeFromBytes(tx)

	mem.txsmx.Lock()
	mem.txs[decodedTx.GasPrice].Remove(elem)
	mem.txsmx.Unlock()

	mem.decrementTxsCounter(decodedTx.GasPrice)

	elem.DetachPrev()
	mem.txsMap.Delete(TxKey(tx))
	atomic.AddInt64(&mem.txsBytes, int64(-len(tx)))

	if removeFromCache {
		mem.cache.Remove(tx)
	}
}

// RemoveTxByKey removes a transaction from the mempool by its TxKey index.
func (mem *PriorityMempool) RemoveTxByKey(txKey [TxKeySize]byte, removeFromCache bool) {
	if e, ok := mem.txsMap.Load(txKey); ok {
		memTx := e.(*clist.CElement).Value.(*mempoolTx)
		if memTx != nil {
			mem.removeTx(memTx.tx, e.(*clist.CElement), removeFromCache)
		}
	}
}

func (mem *PriorityMempool) isFull(txSize int) error {
	var (
		memSize  = mem.Size()
		txsBytes = mem.TxsBytes()
	)

	if memSize >= mem.config.Size || int64(txSize)+txsBytes > mem.config.MaxTxsBytes {
		return ErrMempoolIsFull{
			memSize, mem.config.Size,
			txsBytes, mem.config.MaxTxsBytes,
		}
	}

	return nil
}

// callback, which is called after the app checked the tx for the first time.
//
// The case where the app checks the tx for the second and subsequent times is
// handled by the resCbRecheck callback.
func (mem *PriorityMempool) resCbFirstTime(
	tx []byte,
	peerID uint16,
	peerP2PID p2p.ID,
	res *abci.Response,
) {
	switch r := res.Value.(type) {
	case *abci.Response_CheckTx:
		var postCheckErr error
		if mem.postCheck != nil {
			postCheckErr = mem.postCheck(tx, r.CheckTx)
		}
		if (r.CheckTx.Code == abci.CodeTypeOK) && postCheckErr == nil {
			// Check mempool isn't full again to reduce the chance of exceeding the
			// limits.
			if err := mem.isFull(len(tx)); err != nil {
				// remove from cache (mempool might have a space later)
				mem.cache.Remove(tx)
				mem.logger.Error(err.Error())
				return
			}

			memTx := &mempoolTx{
				height:    mem.height,
				gasWanted: r.CheckTx.GasWanted,
				tx:        tx,
			}
			memTx.senders.Store(peerID, true)
			mem.addTx(memTx)
			mem.logger.Debug("added good transaction",
				"tx", txID(tx),
				"res", r,
				"height", memTx.height,
				"total", mem.Size(),
			)
			mem.notifyTxsAvailable()
		} else {
			// ignore bad transaction
			mem.logger.Debug("rejected bad transaction",
				"tx", txID(tx), "peerID", peerP2PID, "res", r, "err", postCheckErr)
			mem.metrics.FailedTxs.Add(1)
			if !mem.config.KeepInvalidTxsInCache {
				// remove from cache (it might be good later)
				mem.cache.Remove(tx)
			}
		}
	default:
		// ignore other messages
	}
}

// callback, which is called after the app rechecked the tx.
//
// The case where the app checks the tx for the first time is handled by the
// resCbFirstTime callback.
func (mem *PriorityMempool) resCbRecheck(req *abci.Request, res *abci.Response) {
	switch r := res.Value.(type) {
	case *abci.Response_CheckTx:
		tx := req.GetCheckTx().Tx
		memTx := mem.recheckCursor.Value.(*mempoolTx)
		if !bytes.Equal(tx, memTx.tx) {
			panic(fmt.Sprintf(
				"Unexpected tx response from proxy during recheck\nExpected %X, got %X",
				memTx.tx,
				tx))
		}
		var postCheckErr error
		if mem.postCheck != nil {
			postCheckErr = mem.postCheck(tx, r.CheckTx)
		}
		if (r.CheckTx.Code == abci.CodeTypeOK) && postCheckErr == nil {
			// Good, nothing to do.
		} else {
			// Tx became invalidated due to newly committed block.
			mem.logger.Debug("tx is no longer valid", "tx", txID(tx), "res", r, "err", postCheckErr)
			// NOTE: we remove tx from the cache because it might be good later
			mem.removeTx(tx, mem.recheckCursor, !mem.config.KeepInvalidTxsInCache)
		}
		if mem.recheckCursor == mem.recheckEnd {
			mem.recheckCursor = nil
		} else {
			mem.recheckCursor = mem.recheckCursor.Next()
			if mem.recheckCursor == nil {
				tx, _ := mem.executor.DecodeFromBytes(memTx.tx)
				for k, v := range mem.gasPrices {
					if v == tx.GasPrice {
						if k == 0 {
							break
						}

						mem.recheckCursor = mem.getTxByGasPrice(mem.gasPrices[k-1]).Front()
						break
					}
				}
			}
		}
		if mem.recheckCursor == nil {
			// Done!
			mem.logger.Debug("done rechecking txs")

			// incase the recheck removed all txs
			if mem.Size() > 0 {
				mem.notifyTxsAvailable()
			}
		}
	default:
		// ignore other messages
	}
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) TxsAvailable() <-chan struct{} {
	return mem.txsAvailable
}

func (mem *PriorityMempool) notifyTxsAvailable() {
	if mem.Size() == 0 {
		panic("notified txs available but mempool is empty!")
	}
	if mem.txsAvailable != nil && !mem.notifiedTxsAvailable {
		// channel cap is 1, so this will send once
		mem.notifiedTxsAvailable = true
		select {
		case mem.txsAvailable <- struct{}{}:
		default:
		}
	}
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) ReapMaxBytesMaxGas(maxBytes, maxGas int64) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	var totalGas int64

	// TODO: we will get a performance boost if we have a good estimate of avg
	// size per tx, and set the initial capacity based off of that.
	// txs := make([]types.Tx, 0, tmmath.MinInt(mem.txs.Len(), max/mem.avgTxSize))
	gasPrices, txCount := mem.txsByGasPrices()
	txs := make([]types.Tx, 0, txCount)
	for _, gp := range gasPrices {
		for e := mem.getTxByGasPrice(gp).Front(); e != nil; e = e.Next() {
			memTx := e.Value.(*mempoolTx)

			dataSize := types.ComputeProtoSizeForTxs(append(txs, memTx.tx))

			// Check total size requirement
			if maxBytes > -1 && dataSize > maxBytes {
				return txs
			}
			// Check total gas requirement.
			// If maxGas is negative, skip this check.
			// Since newTotalGas < masGas, which
			// must be non-negative, it follows that this won't overflow.
			newTotalGas := totalGas + memTx.gasWanted
			if maxGas > -1 && newTotalGas > maxGas {
				return txs
			}

			totalGas = newTotalGas
			txs = append(txs, memTx.tx)
		}
	}

	return txs
}

// Safe for concurrent use by multiple goroutines.
func (mem *PriorityMempool) ReapMaxTxs(max int) types.Txs {
	mem.updateMtx.RLock()
	defer mem.updateMtx.RUnlock()

	gasPrices, txCount := mem.txsByGasPrices()
	if max < 0 {
		max = int(txCount)
	}

	txs := make([]types.Tx, 0, tmmath.MinInt(int(txCount), max))
	for _, gp := range gasPrices {
		for e := mem.getTxByGasPrice(gp).Front(); e != nil && len(txs) <= max; e = e.Next() {
			memTx := e.Value.(*mempoolTx)
			txs = append(txs, memTx.tx)
		}
	}

	return txs
}

// Lock() must be help by the caller during execution.
func (mem *PriorityMempool) Update(
	height int64,
	txs types.Txs,
	deliverTxResponses []*abci.ResponseDeliverTx,
	preCheck tmpool.PreCheckFunc,
	postCheck tmpool.PostCheckFunc,
) error {
	// Set height
	mem.height = height
	mem.notifiedTxsAvailable = false

	if preCheck != nil {
		mem.preCheck = preCheck
	}
	if postCheck != nil {
		mem.postCheck = postCheck
	}

	for i, tx := range txs {
		if deliverTxResponses[i].Code == abci.CodeTypeOK {
			// Add valid committed tx to the cache (if missing).
			_ = mem.cache.Push(tx)
		} else if !mem.config.KeepInvalidTxsInCache {
			// Allow invalid transactions to be resubmitted.
			mem.cache.Remove(tx)
		}

		// Remove committed tx from the tmpool.
		//
		// Note an evil proposer can drop valid txs!
		// Mempool before:
		//   100 -> 101 -> 102
		// Block, proposed by an evil proposer:
		//   101 -> 102
		// Mempool after:
		//   100
		// https://github.com/tendermint/tendermint/issues/3322.
		if e, ok := mem.txsMap.Load(TxKey(tx)); ok {
			mem.removeTx(tx, e.(*clist.CElement), false)
		}
	}

	// Either recheck non-committed txs to see if they became invalid
	// or just notify there're some txs left.
	if mem.Size() > 0 {
		if mem.config.Recheck {
			mem.logger.Debug("recheck txs", "numtxs", mem.Size(), "height", height)
			mem.recheckTxs()
			// At this point, mem.txs are being rechecked.
			// mem.recheckCursor re-scans mem.txs and possibly removes some txs.
			// Before mem.Reap(), we should wait for mem.recheckCursor to be nil.
		} else {
			mem.notifyTxsAvailable()
		}
	}

	// Update metrics
	mem.metrics.Size.Set(float64(mem.Size()))

	return nil
}

func (mem *PriorityMempool) recheckTxs() {
	if mem.Size() == 0 {
		panic("recheckTxs is called, but the mempool is empty")
	}

	// TODO: handle recheck
	mem.recheckCursor = mem.getTxByGasPrice(mem.gasPrices[len(mem.gasPrices)-1]).Front()
	mem.recheckEnd = mem.getTxByGasPrice(mem.gasPrices[0]).Back()

	// Push txs to proxyAppConn
	// NOTE: globalCb may be called concurrently.
	gasPrices, _ := mem.txsByGasPrices()
	for _, gp := range gasPrices {
		for e := mem.getTxByGasPrice(gp).Front(); e != nil; e = e.Next() {
			memTx := e.Value.(*mempoolTx)
			mem.proxyAppConn.CheckTxAsync(abci.RequestCheckTx{
				Tx:   memTx.tx,
				Type: abci.CheckTxType_Recheck,
			})
		}
	}

	mem.proxyAppConn.FlushAsync()
}

//--------------------------------------------------------------------------------

// mempoolTx is a transaction that successfully ran
type mempoolTx struct {
	height    int64    // height that this tx had been validated in
	gasWanted int64    // amount of gas this tx states it will require
	tx        types.Tx //

	// ids of peers who've sent us this tx (as a map for quick lookups).
	// senders: PeerID -> bool
	senders sync.Map
}

// Height returns the height for this transaction
func (memTx *mempoolTx) Height() int64 {
	return atomic.LoadInt64(&memTx.height)
}
