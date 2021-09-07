package mempool

import (
	"github.com/tendermint/tendermint/abci/example/kvstore"
	tmpool "github.com/tendermint/tendermint/mempool"
	"github.com/tendermint/tendermint/proxy"
	"testing"
)

func BenchmarkReap(b *testing.B) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	size := 10000
	mempool.config.Size = size
	for i := 0; i < size; i++ {
		tx := createTxWithRandomGas(116, nil)
		if err := mempool.CheckTx(tx, nil, tmpool.TxInfo{}); err != nil {
			b.Error(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mempool.ReapMaxBytesMaxGas(100000000, 10000000)
	}
}

func BenchmarkCheckTx(b *testing.B) {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	mempool, cleanup := newMempoolWithApp(cc)
	defer cleanup()

	for i := 0; i < b.N; i++ {
		tx := createTxWithRandomGas(116, nil)
		if err := mempool.CheckTx(tx, nil, tmpool.TxInfo{}); err != nil {
			b.Error(err)
		}
	}
}
