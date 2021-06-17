package mempool

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/helpers"
	"github.com/MinterTeam/minter-go-node/rlp"
	tmpool "github.com/tendermint/tendermint/mempool"
	"io/ioutil"
	"math/big"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MinterTeam/minter-go-node/tests/example/counter"
	"github.com/tendermint/tendermint/abci/example/kvstore"
	abciserver "github.com/tendermint/tendermint/abci/server"
	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"
)

// A cleanupFunc cleans up any config / test files created for a particular
// test.
type cleanupFunc func()

func newMempoolWithApp(cc proxy.ClientCreator) (*PriorityMempool, cleanupFunc) {
	return newMempoolWithAppAndConfig(cc, cfg.ResetTestRoot("mempool_test"))
}

func newMempoolWithAppAndConfig(cc proxy.ClientCreator, config *cfg.Config) (*PriorityMempool, cleanupFunc) {
	appConnMem, _ := cc.NewABCIClient()
	appConnMem.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "mempool"))
	err := appConnMem.Start()
	if err != nil {
		panic(err)
	}
	mempool := NewPriorityMempool(config.Mempool, appConnMem, 0)
	mempool.SetLogger(log.TestingLogger())
	return mempool, func() { os.RemoveAll(config.RootDir) }
}

func ensureNoFire(t *testing.T, ch <-chan struct{}, timeoutMS int) {
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	select {
	case <-ch:
		t.Fatal("Expected not to fire")
	case <-timer.C:
	}
}

func ensureFire(t *testing.T, ch <-chan struct{}, timeoutMS int) {
	timer := time.NewTimer(time.Duration(timeoutMS) * time.Millisecond)
	select {
	case <-ch:
	case <-timer.C:
		t.Fatal("Expected to fire")
	}
}

func createTx(bytesLen uint64, privKey *ecdsa.PrivateKey) []byte {
	encodedData, _ := rlp.EncodeToBytes(transaction.SendData{
		Coin:  types.GetBaseCoinID(),
		To:    [20]byte{1},
		Value: helpers.BipToPip(big.NewInt(10)),
	})

	payload := []byte{0x01, 0x01}
	for i := bytesLen - 116; i > 0; i-- {
		payload = append(payload, 0x01)
	}

	tx := transaction.Transaction{
		Nonce:         uint64(1),
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeSend,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
		Payload:       payload,
	}

	if privKey == nil {
		privKey, _ = crypto.GenerateKey()
	}

	tx.Sign(privKey)
	txBytes, _ := tx.Serialize()
	return txBytes
}

func createTxWithRandomGas(bytesLen uint64, privKey *ecdsa.PrivateKey) []byte {
	encodedData, _ := rlp.EncodeToBytes(transaction.SendData{
		Coin:  types.GetBaseCoinID(),
		To:    [20]byte{1},
		Value: helpers.BipToPip(big.NewInt(10)),
	})

	payload := []byte{0x01, 0x01}
	for i := bytesLen - 116; i > 0; i-- {
		payload = append(payload, 0x01)
	}

	gp := tmrand.Intn(200)
	tx := transaction.Transaction{
		Nonce:         uint64(1),
		ChainID:       types.CurrentChainID,
		GasPrice:      uint32(gp),
		GasCoin:       types.GetBaseCoinID(),
		Type:          transaction.TypeSend,
		Data:          encodedData,
		SignatureType: transaction.SigTypeSingle,
		Payload:       payload,
	}

	if privKey == nil {
		privKey, _ = crypto.GenerateKey()
	}

	tx.Sign(privKey)
	txBytes, _ := tx.Serialize()
	return txBytes
}

func checkTxs(t *testing.T, mempool tmpool.Mempool, count int, peerID uint16) tmtypes.Txs {
	txs := make(tmtypes.Txs, count)
	txInfo := tmpool.TxInfo{SenderID: peerID}
	for i := 0; i < count; i++ {
		txBytes := createTx(116, nil)
		txs[i] = txBytes
		if err := mempool.CheckTx(txBytes, nil, txInfo); err != nil {
			// Skip invalid txs.
			// TestMempoolFilters will fail otherwise. It asserts a number of txs
			// returned.
			if tmpool.IsPreCheckError(err) {
				continue
			}
			t.Fatalf("CheckTx failed: %v while checking #%d tx", err, i)
		}
	}
	return txs
}

func TestReapMaxBytesMaxGas(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	// Ensure gas calculation behaves as expected
	checkTxs(t, mempool, 1, tmpool.UnknownPeerID)
	tx0 := mempool.txs.Front().Value.(*tmpool.MempoolTx)
	//// assert that kv store has gas wanted = 1.
	require.Equal(t, app.CheckTx(abci.RequestCheckTx{Tx: tx0.Tx}).GasWanted, int64(1), "KVStore had a gas value neq to 1")
	require.Equal(t, tx0.GasWanted, int64(1), "transactions gas was set incorrectly")
	//// ensure each tx is 116 bytes long
	require.Equal(t, len(tx0.Tx), 116, "Tx is longer than 116 bytes")
	mempool.Flush()

	// each table driven test creates numTxsToCreate txs with checkTx, and at the end clears all remaining txs.
	// each tx has 20 bytes
	tests := []struct {
		numTxsToCreate int
		maxBytes       int64
		maxGas         int64
		expectedNumTxs int
	}{
		{20, -1, -1, 20},
		{20, -1, 0, 0},
		{20, -1, 10, 10},
		{20, -1, 30, 20},
		{20, 0, -1, 0},
		{20, 0, 10, 0},
		{20, 10, 10, 0},

		{20, 118, 10, 1},
		{20, 590, 5, 5},
		{20, 1180, -1, 10},
		{20, 1180, 10, 10},
		{20, 1180, 15, 10},
		//
		{20, 20000, -1, 20},
		{20, 20000, 5, 5},
		{20, 20000, 30, 20},
	}
	for tcIndex, tt := range tests {
		checkTxs(t, mempool, tt.numTxsToCreate, tmpool.UnknownPeerID)
		got := mempool.ReapMaxBytesMaxGas(tt.maxBytes, tt.maxGas)
		assert.Equal(t, tt.expectedNumTxs, len(got), "Got %d txs, expected %d, tc #%d",
			len(got), tt.expectedNumTxs, tcIndex)
		mempool.Flush()
	}
}

func TestReapMaxBytesMaxGasPriority(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	txInfo := tmpool.TxInfo{SenderID: tmpool.UnknownPeerID}
	for i := 0; i < 1000; i++ {
		if err := mempool.CheckTx(createTxWithRandomGas(116, nil), nil, txInfo); err != nil {
			if tmpool.IsPreCheckError(err) {
				continue
			}
		}
	}

	txs := mempool.ReapMaxBytesMaxGas(10000000000, 10000000000)
	assert.Equal(t, 1000, len(txs))

	executor := transaction.NewExecutor(transaction.GetData)

	for i := 0; i <= len(txs)-2; i++ {
		data, _ := executor.DecodeFromBytes(txs[i])
		data1, _ := executor.DecodeFromBytes(txs[i+1])
		assert.Equal(t, true, data1.GasPrice <= data.GasPrice)
	}
}

func TestMempoolFilters(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	emptyTxArr := []tmtypes.Tx{[]byte{}}

	nopPreFilter := func(tx tmtypes.Tx) error { return nil }
	nopPostFilter := func(tx tmtypes.Tx, res *abci.ResponseCheckTx) error { return nil }

	// each table driven test creates numTxsToCreate txs with checkTx, and at the end clears all remaining txs.
	// each tx has 20 bytes
	tests := []struct {
		numTxsToCreate int
		preFilter      tmpool.PreCheckFunc
		postFilter     tmpool.PostCheckFunc
		expectedNumTxs int
	}{
		{10, nopPreFilter, nopPostFilter, 10},
		{10, tmpool.PreCheckMaxBytes(10), nopPostFilter, 0},
		{10, tmpool.PreCheckMaxBytes(118), nopPostFilter, 10},
		{10, nopPreFilter, tmpool.PostCheckMaxGas(-1), 10},
		{10, nopPreFilter, tmpool.PostCheckMaxGas(0), 0},
		{10, nopPreFilter, tmpool.PostCheckMaxGas(1), 10},
		{10, nopPreFilter, tmpool.PostCheckMaxGas(3000), 10},
		{10, tmpool.PreCheckMaxBytes(10), tmpool.PostCheckMaxGas(20), 0},
		{10, tmpool.PreCheckMaxBytes(126), tmpool.PostCheckMaxGas(116), 10},
		{10, tmpool.PreCheckMaxBytes(118), tmpool.PostCheckMaxGas(1), 10},
		{10, tmpool.PreCheckMaxBytes(118), tmpool.PostCheckMaxGas(0), 0},
	}
	for tcIndex, tt := range tests {
		err := mempool.Update(1, emptyTxArr, abciResponses(len(emptyTxArr), abci.CodeTypeOK), tt.preFilter, tt.postFilter)
		require.NoError(t, err)
		checkTxs(t, mempool, tt.numTxsToCreate, tmpool.UnknownPeerID)
		require.Equal(t, tt.expectedNumTxs, mempool.Size(), "mempool had the incorrect size, on test case %d", tcIndex)
		mempool.Flush()
	}
}

func TestMempoolUpdate(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	// 1. Adds valid txs to the cache
	{
		tx := createTx(116, nil)
		err := mempool.Update(1, []tmtypes.Tx{tx}, abciResponses(1, abci.CodeTypeOK), nil, nil)
		require.NoError(t, err)
		err = mempool.CheckTx(tx, nil, tmpool.TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, tmpool.ErrTxInCache, err)
		}
	}

	// 2. Removes valid txs from the mempool
	{
		tx := createTx(117, nil)
		err := mempool.CheckTx(tx, nil, tmpool.TxInfo{})
		require.NoError(t, err)
		err = mempool.Update(1, []tmtypes.Tx{tx}, abciResponses(1, abci.CodeTypeOK), nil, nil)
		require.NoError(t, err)
		assert.Zero(t, mempool.Size())
	}

	// 3. Removes invalid transactions from the cache and the mempool (if present)
	{
		tx := createTx(118, nil)
		err := mempool.CheckTx(tx, nil, tmpool.TxInfo{})
		require.NoError(t, err)
		err = mempool.Update(1, []tmtypes.Tx{tx}, abciResponses(1, 1), nil, nil)
		require.NoError(t, err)
		assert.Zero(t, mempool.Size())

		err = mempool.CheckTx(tx, nil, tmpool.TxInfo{})
		require.NoError(t, err)
	}
}

func TestMempool_KeepInvalidTxsInCache(t *testing.T) {
	app := counter.NewApplication(true, nil)
	cc := proxy.NewLocalClientCreator(app)
	wcfg := cfg.DefaultConfig()
	wcfg.Mempool.KeepInvalidTxsInCache = true
	mempool, cleanup := newMempoolWithAppAndConfig(cc, wcfg)
	defer cleanup()

	// 1. An invalid transaction must remain in the cache after Update
	{
		a := createTx(116, nil)
		b := createTx(116, nil)

		err := mempool.CheckTx(b, nil, tmpool.TxInfo{})
		require.NoError(t, err)

		// simulate new block
		_ = app.DeliverTx(abci.RequestDeliverTx{Tx: a})
		_ = app.DeliverTx(abci.RequestDeliverTx{Tx: b})
		err = mempool.Update(1, []tmtypes.Tx{a, b}, []*abci.ResponseDeliverTx{{Code: abci.CodeTypeOK}, {Code: 2}}, nil, nil)
		require.NoError(t, err)

		// a must be added to the cache
		err = mempool.CheckTx(a, nil, tmpool.TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, tmpool.ErrTxInCache, err)
		}

		// b must remain in the cache
		err = mempool.CheckTx(b, nil, tmpool.TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, tmpool.ErrTxInCache, err)
		}
	}

	// 2. An invalid transaction must remain in the cache
	{
		a := createTx(116, nil)

		// remove a from the cache to test (2)
		mempool.cache.Remove(a)

		err := mempool.CheckTx(a, nil, tmpool.TxInfo{})
		require.NoError(t, err)

		err = mempool.CheckTx(a, nil, tmpool.TxInfo{})
		if assert.Error(t, err) {
			assert.Equal(t, tmpool.ErrTxInCache, err)
		}
	}
}

func TestTxsAvailable(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()
	mempool.EnableTxsAvailable()

	timeoutMS := 500

	// with no txs, it shouldnt fire
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch of txs, it should only fire once
	txs := checkTxs(t, mempool, 100, tmpool.UnknownPeerID)
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// call update with half the txs.
	// it should fire once now for the new height
	// since there are still txs left
	committedTxs, txs := txs[:50], txs[50:]
	if err := mempool.Update(1, committedTxs, abciResponses(len(committedTxs), abci.CodeTypeOK), nil, nil); err != nil {
		t.Error(err)
	}
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch more txs. we already fired for this height so it shouldnt fire again
	moreTxs := checkTxs(t, mempool, 50, tmpool.UnknownPeerID)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// now call update with all the txs. it should not fire as there are no txs left
	committedTxs = append(txs, moreTxs...) //nolint: gocritic
	if err := mempool.Update(2, committedTxs, abciResponses(len(committedTxs), abci.CodeTypeOK), nil, nil); err != nil {
		t.Error(err)
	}
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)

	// send a bunch more txs, it should only fire once
	checkTxs(t, mempool, 100, tmpool.UnknownPeerID)
	ensureFire(t, mempool.TxsAvailable(), timeoutMS)
	ensureNoFire(t, mempool.TxsAvailable(), timeoutMS)
}

func TestSerialReap(t *testing.T) {
	app := counter.NewApplication(true, nil)
	app.SetOption(abci.RequestSetOption{Key: "serial", Value: "on"})
	cc := proxy.NewLocalClientCreator(app)

	mempool, cleanup := newMempoolWithApp(cc)
	fmt.Println(mempool.config)
	defer cleanup()

	appConnCon, _ := cc.NewABCIClient()
	appConnCon.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "consensus"))
	err := appConnCon.Start()
	require.Nil(t, err)

	priv, _ := crypto.GenerateKey()

	cacheMap := make(map[string]struct{})
	deliverTxsRange := func(start, end int) {
		// Deliver some txs.
		for i := start; i < end; i++ {
			// This will succeeds
			txBytes := createTx(116+uint64(i), priv)
			err := mempool.CheckTx(txBytes, nil, tmpool.TxInfo{})
			_, cached := cacheMap[string(txBytes)]
			if cached {
				require.NotNil(t, err, "expected error for cached tx")
			} else {
				require.Nil(t, err, "expected no err for uncached tx")
			}
			cacheMap[string(txBytes)] = struct{}{}

			// Duplicates are cached and should return error
			err = mempool.CheckTx(txBytes, nil, tmpool.TxInfo{})
			require.NotNil(t, err, "Expected error after CheckTx on duplicated tx")
		}
	}

	reapCheck := func(exp int) {
		txs := mempool.ReapMaxBytesMaxGas(-1, -1)
		require.Equal(t, len(txs), exp, fmt.Sprintf("Expected to reap %v txs but got %v", exp, len(txs)))
	}

	updateRange := func(start, end int) {
		txs := make([]tmtypes.Tx, 0)
		for i := start; i < end; i++ {
			txBytes := createTx(116+uint64(i), priv)
			txs = append(txs, txBytes)
		}
		if err := mempool.Update(0, txs, abciResponses(len(txs), abci.CodeTypeOK), nil, nil); err != nil {
			t.Error(err)
		}
	}

	commitRange := func(start, end int) {
		// Deliver some txs.
		for i := start; i < end; i++ {
			txBytes := createTx(116+uint64(i), priv)
			res, err := appConnCon.DeliverTxSync(abci.RequestDeliverTx{Tx: txBytes})
			if err != nil {
				t.Errorf("client error committing tx: %v", err)
			}
			if res.IsErr() {
				t.Errorf("error committing tx. Code:%v result:%X log:%v",
					res.Code, res.Data, res.Log)
			}
		}
		res, err := appConnCon.CommitSync()
		if err != nil {
			t.Errorf("client error committing: %v", err)
		}
		if len(res.Data) != 8 {
			t.Errorf("error committing. Hash:%X", res.Data)
		}
	}

	//----------------------------------------

	// Deliver some txs.
	deliverTxsRange(0, 100)

	// Reap the txs.
	reapCheck(100)

	// Reap again.  We should get the same amount
	reapCheck(100)

	// Deliver 0 to 999, we should reap 900 new txs
	// because 100 were already counted.
	deliverTxsRange(0, 1000)

	// Reap the txs.
	reapCheck(1000)

	// Reap again.  We should get the same amount
	reapCheck(1000)

	// Commit from the conensus AppConn
	commitRange(0, 500)
	updateRange(0, 500)

	// We should have 500 left.
	reapCheck(500)

	// Deliver 100 invalid txs and 100 valid txs
	deliverTxsRange(900, 1100)

	// We should have 600 now.
	reapCheck(600)
}

func TestMempoolCloseWAL(t *testing.T) {
	// 1. Create the temporary directory for mempool and WAL testing.
	rootDir, err := ioutil.TempDir("", "mempool-test")
	require.Nil(t, err, "expecting successful tmpdir creation")

	// 2. Ensure that it doesn't contain any elements -- Sanity check
	m1, err := filepath.Glob(filepath.Join(rootDir, "*"))
	require.Nil(t, err, "successful globbing expected")
	require.Equal(t, 0, len(m1), "no matches yet")

	// 3. Create the mempool
	wcfg := cfg.DefaultConfig()
	wcfg.Mempool.RootDir = rootDir
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithAppAndConfig(cc, wcfg)
	defer cleanup()
	mempool.height = 10
	err = mempool.InitWAL()
	require.NoError(t, err)

	// 4. Ensure that the directory contains the WAL file
	m2, err := filepath.Glob(filepath.Join(rootDir, "*"))
	require.Nil(t, err, "successful globbing expected")
	require.Equal(t, 1, len(m2), "expecting the wal match in")

	// 5. Write some contents to the WAL
	tx := createTx(116, nil)
	err = mempool.CheckTx(tx, nil, tmpool.TxInfo{})
	require.NoError(t, err)
	walFilepath := mempool.wal.Path
	sum1 := checksumFile(walFilepath, t)

	//// 6. Sanity check to ensure that the written TX matches the expectation.
	require.Equal(t, sum1, checksumIt(append(tx, []byte("\n")...)), "foo with a newline should be written")

	// 7. Invoke CloseWAL() and ensure it discards the
	// WAL thus any other write won't go through.
	mempool.CloseWAL()
	err = mempool.CheckTx(createTx(117, nil), nil, tmpool.TxInfo{})
	require.NoError(t, err)
	sum2 := checksumFile(walFilepath, t)
	require.Equal(t, sum1, sum2, "expected no change to the WAL after invoking CloseWAL() since it was discarded")

	// 8. Sanity check to ensure that the WAL file still exists
	m3, err := filepath.Glob(filepath.Join(rootDir, "*"))
	require.Nil(t, err, "successful globbing expected")
	require.Equal(t, 1, len(m3), "expecting the wal match in")
}

func TestMempool_CheckTxChecksTxSize(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempl, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	maxTxSize := mempl.config.MaxTxBytes

	testCases := []struct {
		len    int
		err    bool
		experr int
	}{
		// check small txs. no error
		0: {10, false, 0},
		1: {1000, false, 0},
		2: {1000000, false, 0},

		// check around maxTxSize
		3: {maxTxSize - 122, false, 0},
		4: {maxTxSize - 121, false, 0},
		5: {maxTxSize - 120, true, maxTxSize + 1},
	}

	for i, testCase := range testCases {
		caseString := fmt.Sprintf("case %d, len %d", i, testCase.len)

		tx := createTx(uint64(116+testCase.len), nil)

		err := mempl.CheckTx(tx, nil, tmpool.TxInfo{})
		bv := gogotypes.BytesValue{Value: tx}
		bz, err2 := bv.Marshal()
		require.NoError(t, err2)
		require.Equal(t, len(bz), proto.Size(&bv), caseString)

		if !testCase.err {
			require.NoError(t, err, caseString)
		} else {
			require.Equal(t, err, ErrTxTooLarge{maxTxSize, testCase.experr}, caseString)
		}
	}
}

func TestMempoolTxsBytes(t *testing.T) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	config := cfg.ResetTestRoot("mempool_test")
	config.Mempool.MaxTxsBytes = 126
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	// 1. zero by default
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 2. len(tx) after CheckTx
	tx1 := createTx(116, nil)
	err := mempool.CheckTx(tx1, nil, tmpool.TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 116, mempool.TxsBytes())

	// 3. zero again after tx is removed by Update
	err = mempool.Update(1, []tmtypes.Tx{tx1}, abciResponses(1, abci.CodeTypeOK), nil, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 4. zero after Flush
	err = mempool.CheckTx(createTx(117, nil), nil, tmpool.TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 117, mempool.TxsBytes())

	mempool.Flush()
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 5. ErrMempoolIsFull is returned when/if MaxTxsBytes limit is reached.
	err = mempool.CheckTx(createTx(126, nil), nil, tmpool.TxInfo{})
	require.NoError(t, err)
	err = mempool.CheckTx(createTx(117, nil), nil, tmpool.TxInfo{})
	if assert.Error(t, err) {
		assert.IsType(t, ErrMempoolIsFull{}, err)
	}

	// 6. zero after tx is rechecked and removed due to not being valid anymore
	app2 := counter.NewApplication(true, transaction.NewExecutor(transaction.GetData))
	cc = proxy.NewLocalClientCreator(app2)
	mempool, cleanup = newMempoolWithApp(cc)
	defer cleanup()

	app2.TxCount = 1
	tx := createTx(124, nil)
	err = mempool.CheckTx(tx, nil, tmpool.TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 124, mempool.TxsBytes())

	appConnCon, _ := cc.NewABCIClient()
	appConnCon.SetLogger(log.TestingLogger().With("module", "abci-client", "connection", "consensus"))
	err = appConnCon.Start()
	require.Nil(t, err)
	t.Cleanup(func() {
		if err := appConnCon.Stop(); err != nil {
			t.Error(err)
		}
	})
	res, err := appConnCon.DeliverTxSync(abci.RequestDeliverTx{Tx: tx})
	require.NoError(t, err)
	require.EqualValues(t, uint32(0), res.Code)
	res2, err := appConnCon.CommitSync()
	require.NoError(t, err)
	require.NotEmpty(t, res2.Data)

	// Pretend like we committed nothing so txBytes gets rechecked and removed.
	err = mempool.Update(1, []tmtypes.Tx{}, abciResponses(0, abci.CodeTypeOK), nil, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 0, mempool.TxsBytes())

	// 7. Test RemoveTxByK	ey function
	mempool, cleanup = newMempoolWithAppAndConfig(proxy.NewLocalClientCreator(app), config)
	defer cleanup()

	tx = createTx(123, nil)
	err = mempool.CheckTx(tx, nil, tmpool.TxInfo{})
	require.NoError(t, err)
	assert.EqualValues(t, 123, mempool.TxsBytes())
	mempool.RemoveTxByKey(TxKey(createTx(124, nil)), true)
	assert.EqualValues(t, 123, mempool.TxsBytes())
	mempool.RemoveTxByKey(TxKey(tx), true)
	assert.EqualValues(t, 0, mempool.TxsBytes())

}

// This will non-deterministically catch some concurrency failures like
// https://github.com/tendermint/tendermint/issues/3509
// TODO: all of the tests should probably also run using the remote proxy app
// since otherwise we're not actually testing the concurrency of the mempool here!
func TestMempoolRemoteAppConcurrency(t *testing.T) {
	sockPath := fmt.Sprintf("unix:///tmp/echo_%v.sock", tmrand.Str(6))
	app := kvstore.NewApplication()
	cc, server := newRemoteApp(t, sockPath, app)
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Error(err)
		}
	})
	config := cfg.ResetTestRoot("mempool_test")
	mempool, cleanup := newMempoolWithAppAndConfig(cc, config)
	defer cleanup()

	// generate small number of txs
	nTxs := 10
	txs := make([]tmtypes.Tx, nTxs)
	for i := 0; i < nTxs; i++ {
		txs[i] = createTx(116, nil)
	}

	// simulate a group of peers sending them over and over
	N := config.Mempool.Size
	maxPeers := 5
	for i := 0; i < N; i++ {
		peerID := mrand.Intn(maxPeers)
		txNum := mrand.Intn(nTxs)
		tx := txs[txNum]

		// this will err with tmpool.ErrTxInCache many times ...
		mempool.CheckTx(tx, nil, tmpool.TxInfo{SenderID: uint16(peerID)}) //nolint: errcheck // will error
	}
	err := mempool.FlushAppConn()
	require.NoError(t, err)
}

// caller must close server
func newRemoteApp(
	t *testing.T,
	addr string,
	app abci.Application,
) (
	clientCreator proxy.ClientCreator,
	server service.Service,
) {
	clientCreator = proxy.NewRemoteClientCreator(addr, "socket", true)

	// Start server
	server = abciserver.NewSocketServer(addr, app)
	server.SetLogger(log.TestingLogger().With("module", "abci-server"))
	if err := server.Start(); err != nil {
		t.Fatalf("Error starting socket server: %v", err.Error())
	}
	return clientCreator, server
}
func checksumIt(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func checksumFile(p string, t *testing.T) string {
	data, err := ioutil.ReadFile(p)
	require.Nil(t, err, "expecting successful read of %q", p)
	return checksumIt(data)
}

func abciResponses(n int, code uint32) []*abci.ResponseDeliverTx {
	responses := make([]*abci.ResponseDeliverTx, 0, n)
	for i := 0; i < n; i++ {
		responses = append(responses, &abci.ResponseDeliverTx{Code: code})
	}
	return responses
}
