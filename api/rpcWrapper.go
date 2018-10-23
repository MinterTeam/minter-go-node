package api

import (
	rpc "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"sync"
)

type Local struct {
	client *rpc.Local

	mtx sync.Mutex
}

func (c *Local) Status() (*ctypes.ResultStatus, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Status()
}

func (c *Local) ABCIInfo() (*ctypes.ResultABCIInfo, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.ABCIInfo()
}

func (c *Local) BroadcastTxCommit(tx types.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.BroadcastTxCommit(tx)
}

func (c *Local) BroadcastTxAsync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.BroadcastTxAsync(tx)
}

func (c *Local) BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.BroadcastTxSync(tx)
}

func (c *Local) NetInfo() (*ctypes.ResultNetInfo, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.NetInfo()
}

func (c *Local) DumpConsensusState() (*ctypes.ResultDumpConsensusState, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.DumpConsensusState()
}

func (c *Local) ConsensusState() (*ctypes.ResultConsensusState, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.ConsensusState()
}

func (c *Local) Health() (*ctypes.ResultHealth, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Health()
}

func (c *Local) BlockchainInfo(minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.BlockchainInfo(minHeight, maxHeight)
}

func (c *Local) Genesis() (*ctypes.ResultGenesis, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Genesis()
}

func (c *Local) Block(height *int64) (*ctypes.ResultBlock, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Block(height)
}

func (c *Local) BlockResults(height *int64) (*ctypes.ResultBlockResults, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.BlockResults(height)
}

func (c *Local) Commit(height *int64) (*ctypes.ResultCommit, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Commit(height)
}

func (c *Local) Validators(height *int64) (*ctypes.ResultValidators, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Validators(height)
}

func (c *Local) Tx(hash []byte, prove bool) (*ctypes.ResultTx, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.Tx(hash, prove)
}

func (c *Local) TxSearch(query string, prove bool, page, perPage int) (*ctypes.ResultTxSearch, error) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.client.TxSearch(query, prove, page, perPage)
}
