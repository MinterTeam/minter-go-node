package api

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"minter/core/minter"
	"minter/rpc/lib/client"
	"minter/tmtypes"
	"time"
	"minter/cmd/utils"
)

var (
	blockchain        *minter.Blockchain
	tendermintRpcAddr = utils.TendermintRpcAddrFlag.Value
)

func RunApi(b *minter.Blockchain) {
	blockchain = b

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/api/balance/{address}", GetBalance).Methods("GET")
	router.HandleFunc("/api/balanceWS", GetBalanceWatcher)
	router.HandleFunc("/api/transactionCount/{address}", GetTransactionCount).Methods("GET")
	router.HandleFunc("/api/sendTransaction", SendTransaction).Methods("POST")
	router.HandleFunc("/api/transaction/{hash}", Transaction).Methods("GET")
	router.HandleFunc("/api/block/{height}", Block).Methods("GET")
	router.HandleFunc("/api/transactions", Transactions).Methods("GET")
	router.HandleFunc("/api/status", Status).Methods("GET")
	router.HandleFunc("/api/coinInfo/{symbol}", GetCoinInfo).Methods("GET")
	router.HandleFunc("/api/estimateCoinExchangeReturn", EstimateCoinExchangeReturn).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"POST", "GET"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	client := rpcclient.NewJSONRPCClient(tendermintRpcAddr)
	tmtypes.RegisterAmino(client.Codec())

	// wait for tendermint to start
	for true {
		result := new(tmtypes.ResultStatus)
		_, err := client.Call("status", map[string]interface{}{}, result)
		if err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}

	log.Fatal(http.ListenAndServe(":8841", handler))
}

type Response struct {
	Code   uint32      `json:"code"`
	Result interface{} `json:"result"`
	Log    string      `json:"log,omitempty"`
}
