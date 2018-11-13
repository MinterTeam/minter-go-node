package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/encoding/amino"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"strconv"
	"time"
)

var (
	cdc         = amino.NewCodec()
	blockchain  *minter.Blockchain
	client      *rpc.Local
	connections = int32(0)
	limitter    = make(chan struct{}, 10)
	cfg         = config.GetConfig()
)

func init() {
	cryptoAmino.RegisterAmino(cdc)
	eventsdb.RegisterAminoEvents(cdc)
}

func RunApi(b *minter.Blockchain, tmRPC *rpc.Local) {
	client = tmRPC

	blockchain = b

	router := mux.NewRouter().StrictSlash(true)

	stats := IpStats{
		ips:  make(map[string]int),
		lock: sync.Mutex{},
	}

	router.Use(RateLimit(cfg.APIPerIPLimit, cfg.APIPerIPLimitWindow, &stats))

	router.HandleFunc("/api/candidates", wrapper(GetCandidates)).Methods("GET")
	router.HandleFunc("/api/candidate/{pubkey}", wrapper(GetCandidate)).Methods("GET")
	router.HandleFunc("/api/validators", wrapper(GetValidators)).Methods("GET")
	router.HandleFunc("/api/balance/{address}", wrapper(GetBalance)).Methods("GET")
	router.HandleFunc("/api/transactionCount/{address}", wrapper(GetTransactionCount)).Methods("GET")
	router.HandleFunc("/api/sendTransaction", wrapper(SendTransaction)).Methods("POST")
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

	log.Error("Failed to start API", "err", http.ListenAndServe(config.GetConfig().APIListenAddress, handler))
}

func wrapper(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&connections) > int32(cfg.APISimultaneousRequests) {
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(Response{
				Code: http.StatusTooManyRequests,
				Log:  "Too many requests",
			})
			return
		}

		atomic.AddInt32(&connections, 1)
		limitter <- struct{}{}

		defer func() {
			log.With("module", "api").Info("Served API request", "req", r.RequestURI)
			<-limitter
			atomic.AddInt32(&connections, -1)
		}()

		f(w, r)
	}
}

type IpStats struct {
	ips  map[string]int
	lock sync.Mutex
}

func (s *IpStats) Reset() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ips = make(map[string]int)
}

func (s *IpStats) Add(identifier string, count int) int {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ips[identifier] += count

	return s.ips[identifier]
}

type Stats interface {
	// Reset will reset the map.
	Reset()

	// Add would add "count" to the map at the key of "identifier",
	// and returns an int which is the total count of the value
	// at that key.
	Add(identifier string, count int) int
}

func RateLimit(limit int, window time.Duration, stats Stats) func(next http.Handler) http.Handler {
	var windowStart time.Time

	// Clear the rate limit stats after each window.
	ticker := time.NewTicker(window)
	go func() {
		windowStart = time.Now()

		for range ticker.C {
			windowStart = time.Now()
			stats.Reset()
		}
	}()

	return func(next http.Handler) http.Handler {
		h := func(w http.ResponseWriter, r *http.Request) {
			value := int(stats.Add(identifyRequest(r), 1))

			XRateLimitRemaining := limit - value
			if XRateLimitRemaining < 0 {
				XRateLimitRemaining = 0
			}

			w.Header().Add("X-Rate-Limit-Limit", strconv.Itoa(limit))
			w.Header().Add("X-Rate-Limit-Remaining", strconv.Itoa(XRateLimitRemaining))
			w.Header().Add("X-Rate-Limit-Reset", strconv.Itoa(int(window.Seconds()-time.Since(windowStart).Seconds())+1))

			if value >= limit {
				w.WriteHeader(429)
			} else {
				next.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(h)
	}
}

func identifyRequest(r *http.Request) string {
	return strings.Split(r.Header.Get("X-Real-IP"), ":")[0]
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

func GetStateForRequest(r *http.Request) (*state.StateDB, error) {
	height, _ := strconv.Atoi(r.URL.Query().Get("height"))

	if height > 0 {
		cState, err := blockchain.GetStateForHeight(height)

		return cState, err
	}

	return blockchain.CurrentState(), nil
}
