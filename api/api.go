package api

import (
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	"github.com/tendermint/tendermint/node"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"strconv"
	"time"
)

var (
	cdc        = amino.NewCodec()
	blockchain *minter.Blockchain
	client     *rpc.Local
)

func init() {
	cryptoAmino.RegisterAmino(cdc)
	eventsdb.RegisterAminoEvents(cdc)
}

func RunApi(b *minter.Blockchain, node *node.Node) {
	client = rpc.NewLocal(node)

	blockchain = b

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/api/bipVolume", wrapper(GetBipVolume)).Methods("GET")
	router.HandleFunc("/api/candidates", wrapper(GetCandidates)).Methods("GET")
	router.HandleFunc("/api/candidate/{pubkey}", wrapper(GetCandidate)).Methods("GET")
	router.HandleFunc("/api/validators", wrapper(GetValidators)).Methods("GET")
	router.HandleFunc("/api/balance/{address}", wrapper(GetBalance)).Methods("GET")
	router.HandleFunc("/api/balanceWS", wrapper(GetBalanceWatcher))
	router.HandleFunc("/api/transactionCount/{address}", wrapper(GetTransactionCount)).Methods("GET")
	router.HandleFunc("/api/sendTransaction", wrapper(SendTransaction)).Methods("POST")
	router.HandleFunc("/api/sendTransactionSync", wrapper(SendTransactionSync)).Methods("POST")
	router.HandleFunc("/api/sendTransactionAsync", wrapper(SendTransactionAsync)).Methods("POST")
	router.HandleFunc("/api/transaction/{hash}", wrapper(Transaction)).Methods("GET")
	router.HandleFunc("/api/block/{height}", wrapper(Block)).Methods("GET")
	router.HandleFunc("/api/transactions", wrapper(Transactions)).Methods("GET")
	router.HandleFunc("/api/status", wrapper(Status)).Methods("GET")
	router.HandleFunc("/api/net_info", wrapper(NetInfo)).Methods("GET")
	router.HandleFunc("/api/coinInfo/{symbol}", wrapper(GetCoinInfo)).Methods("GET")
	router.HandleFunc("/api/estimateCoinSell", wrapper(EstimateCoinSell)).Methods("GET")
	router.HandleFunc("/api/estimateCoinBuy", wrapper(EstimateCoinBuy)).Methods("GET")
	router.HandleFunc("/api/estimateTxCommission", wrapper(EstimateTxCommission)).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// wait for tendermint to start
	waitForTendermint()

	log.Fatal(http.ListenAndServe(config.GetConfig().APIListenAddress, handler))
}

func wrapper(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer io.Copy(ioutil.Discard, r.Body)
		defer r.Body.Close()

		f(w, r)
	}
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

func GetStateForRequest(r *http.Request) *state.StateDB {
	height, _ := strconv.Atoi(r.URL.Query().Get("height"))

	cState := blockchain.CurrentState()

	if height > 0 {
		cState, _ = blockchain.GetStateForHeight(height)
	}

	return cState
}
