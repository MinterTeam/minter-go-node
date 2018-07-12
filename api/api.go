package api

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"strconv"
	"time"
)

var (
	blockchain *minter.Blockchain
	client     *rpc.HTTP
)

func RunApi(b *minter.Blockchain) {
	client = rpc.NewHTTP(*utils.TendermintRpcAddrFlag, "/websocket")

	blockchain = b

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/api/bipVolume", GetBipVolume).Methods("GET")
	router.HandleFunc("/api/candidate/{pubkey}", GetCandidate).Methods("GET")
	router.HandleFunc("/api/validators", GetValidators).Methods("GET")
	router.HandleFunc("/api/balance/{address}", GetBalance).Methods("GET")
	router.HandleFunc("/api/balanceWS", GetBalanceWatcher)
	router.HandleFunc("/api/transactionCount/{address}", GetTransactionCount).Methods("GET")
	router.HandleFunc("/api/sendTransaction", SendTransaction).Methods("POST")
	router.HandleFunc("/api/sendTransactionSync", SendTransactionSync).Methods("POST")
	router.HandleFunc("/api/sendTransactionAsync", SendTransactionAsync).Methods("POST")
	router.HandleFunc("/api/transaction/{hash}", Transaction).Methods("GET")
	router.HandleFunc("/api/block/{height}", Block).Methods("GET")
	router.HandleFunc("/api/transactions", Transactions).Methods("GET")
	router.HandleFunc("/api/status", Status).Methods("GET")
	router.HandleFunc("/api/coinInfo/{symbol}", GetCoinInfo).Methods("GET")
	router.HandleFunc("/api/estimateCoinSell", EstimateCoinSell).Methods("GET")
	router.HandleFunc("/api/estimateCoinBuy", EstimateCoinBuy).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	// wait for tendermint to start
	for true {
		_, err := client.Health()
		if err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}

	log.Fatal(http.ListenAndServe(*utils.MinterAPIAddrFlag, handler))
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
