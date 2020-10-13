package api

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/rpc/lib/server"
	"github.com/rs/cors"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/libs/log"
	rpc "github.com/tendermint/tendermint/rpc/client/local"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	cdc        = amino.NewCodec()
	blockchain *minter.Blockchain
	client     *rpc.Local
	minterCfg  *config.Config
)

var Routes = map[string]*rpcserver.RPCFunc{
	"status":                 rpcserver.NewRPCFunc(Status, ""),
	"candidates":             rpcserver.NewRPCFunc(Candidates, "height,include_stakes"),
	"candidate":              rpcserver.NewRPCFunc(Candidate, "pub_key,height"),
	"validators":             rpcserver.NewRPCFunc(Validators, "height"),
	"address":                rpcserver.NewRPCFunc(Address, "address,height"),
	"addresses":              rpcserver.NewRPCFunc(Addresses, "addresses,height"),
	"send_transaction":       rpcserver.NewRPCFunc(SendTransaction, "tx"),
	"transaction":            rpcserver.NewRPCFunc(Transaction, "hash"),
	"transactions":           rpcserver.NewRPCFunc(Transactions, "query,page,perPage"),
	"block":                  rpcserver.NewRPCFunc(Block, "height"),
	"events":                 rpcserver.NewRPCFunc(Events, "height"),
	"net_info":               rpcserver.NewRPCFunc(NetInfo, ""),
	"coin_info":              rpcserver.NewRPCFunc(CoinInfo, "symbol,id,height"),
	"estimate_coin_sell":     rpcserver.NewRPCFunc(EstimateCoinSell, "coin_to_sell,coin_to_buy,value_to_sell,height"),
	"estimate_coin_sell_all": rpcserver.NewRPCFunc(EstimateCoinSellAll, "coin_to_sell,coin_to_buy,value_to_sell,height"),
	"estimate_coin_buy":      rpcserver.NewRPCFunc(EstimateCoinBuy, "coin_to_sell,coin_to_buy,value_to_buy,height"),
	"estimate_tx_commission": rpcserver.NewRPCFunc(EstimateTxCommission, "tx,height"),
	"unconfirmed_txs":        rpcserver.NewRPCFunc(UnconfirmedTxs, "limit"),
	"max_gas":                rpcserver.NewRPCFunc(MaxGas, "height"),
	"min_gas_price":          rpcserver.NewRPCFunc(MinGasPrice, ""),
	"genesis":                rpcserver.NewRPCFunc(Genesis, ""),
	"missed_blocks":          rpcserver.NewRPCFunc(MissedBlocks, "pub_key,height"),
	"waitlist":               rpcserver.NewRPCFunc(Waitlist, "pub_key,address,height"),
}

func responseTime(b *minter.Blockchain) func(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			f(w, r)
			go b.StatisticData().SetApiTime(time.Since(start), r.URL.Path)
		}
	}
}

// RunAPI start
func RunAPI(codec *amino.Codec, b *minter.Blockchain, tmRPC *rpc.Local, cfg *config.Config, logger log.Logger) {
	cdc = codec
	minterCfg = cfg
	client = tmRPC
	blockchain = b
	waitForTendermint()

	m := http.NewServeMux()

	rpcserver.RegisterRPCFuncs(m, Routes, cdc, logger.With("module", "rpc"), responseTime(b))
	listener, err := rpcserver.Listen(cfg.APIListenAddress, rpcserver.Config{
		MaxOpenConnections: cfg.APISimultaneousRequests,
	})

	if err != nil {
		panic(err)
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET"},
		AllowCredentials: true,
	})

	handler := c.Handler(m)
	logger.Error("Failed to start API", "err", rpcserver.StartHTTPServer(listener, Handler(handler), logger))
}

func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		for key, value := range query {
			val := value[0]
			if strings.HasPrefix(val, "Mx") {
				query.Set(key, fmt.Sprintf("\"%s\"", val))
			}

			if strings.HasPrefix(val, "Mp") {
				query.Set(key, fmt.Sprintf("\"%s\"", val))
			}
		}

		var err error
		r.URL, err = url.ParseRequestURI(fmt.Sprintf("%s?%s", r.URL.Path, query.Encode()))
		if err != nil {
			panic(err)
		}

		h.ServeHTTP(w, r)
	})
}

func waitForTendermint() {
	for {
		_, err := client.Health()
		if err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

type Response struct {
	Code   uint32      `json:"code"`
	Result interface{} `json:"result,omitempty"`
	Log    string      `json:"log,omitempty"`
}

func GetStateForHeight(height int) (*state.CheckState, error) {
	if height > 0 {
		cState, err := blockchain.GetStateForHeight(uint64(height))

		return cState, err
	}

	return blockchain.CurrentState(), nil
}
