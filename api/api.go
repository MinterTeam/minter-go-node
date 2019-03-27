package api

import (
	"fmt"
	"github.com/MinterTeam/go-amino"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/eventsdb"
	"github.com/MinterTeam/minter-go-node/log"
	"github.com/MinterTeam/minter-go-node/rpc/lib/server"
	"github.com/rs/cors"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/multisig"
	"github.com/tendermint/tendermint/crypto/secp256k1"
	"github.com/tendermint/tendermint/evidence"
	rpc "github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	cdc        = amino.NewCodec()
	blockchain *minter.Blockchain
	client     *rpc.Local
	cfg        = config.GetConfig()
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
	"transactions":           rpcserver.NewRPCFunc(Transactions, "query"),
	"block":                  rpcserver.NewRPCFunc(Block, "height"),
	"events":                 rpcserver.NewRPCFunc(Events, "height"),
	"net_info":               rpcserver.NewRPCFunc(NetInfo, ""),
	"coin_info":              rpcserver.NewRPCFunc(CoinInfo, "symbol,height"),
	"estimate_coin_sell":     rpcserver.NewRPCFunc(EstimateCoinSell, "coin_to_sell,coin_to_buy,value_to_sell,height"),
	"estimate_coin_buy":      rpcserver.NewRPCFunc(EstimateCoinBuy, "coin_to_sell,coin_to_buy,value_to_buy,height"),
	"estimate_tx_commission": rpcserver.NewRPCFunc(EstimateTxCommission, "tx,height"),
	"unconfirmed_txs":        rpcserver.NewRPCFunc(UnconfirmedTxs, "limit"),
	"max_gas":                rpcserver.NewRPCFunc(MaxGas, "height"),
	"min_gas_price":          rpcserver.NewRPCFunc(MinGasPrice, ""),
}

func RunAPI(b *minter.Blockchain, tmRPC *rpc.Local) {
	RegisterCryptoAmino(cdc)
	eventsdb.RegisterAminoEvents(cdc)
	RegisterEvidenceMessages(cdc)

	client = tmRPC
	blockchain = b
	waitForTendermint()

	m := http.NewServeMux()
	logger := log.With("module", "rpc")
	rpcserver.RegisterRPCFuncs(m, Routes, cdc, logger)
	listener, err := rpcserver.Listen(config.GetConfig().APIListenAddress, rpcserver.Config{
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
	log.Error("Failed to start API", "err", rpcserver.StartHTTPServer(listener, Handler(handler), logger))
}

func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()

		for key, value := range query {
			val := value[0]
			if strings.HasPrefix(val, "Mx") {
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

func GetStateForHeight(height int) (*state.StateDB, error) {
	if height > 0 {
		cState, err := blockchain.GetStateForHeight(uint64(height))

		return cState, err
	}

	return blockchain.CurrentState(), nil
}

// RegisterAmino registers all crypto related types in the given (amino) codec.
func RegisterCryptoAmino(cdc *amino.Codec) {
	// These are all written here instead of
	cdc.RegisterInterface((*crypto.PubKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PubKeyEd25519{},
		ed25519.PubKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PubKeySecp256k1{},
		secp256k1.PubKeyAminoName, nil)
	cdc.RegisterConcrete(multisig.PubKeyMultisigThreshold{},
		multisig.PubKeyMultisigThresholdAminoRoute, nil)

	cdc.RegisterInterface((*crypto.PrivKey)(nil), nil)
	cdc.RegisterConcrete(ed25519.PrivKeyEd25519{},
		ed25519.PrivKeyAminoName, nil)
	cdc.RegisterConcrete(secp256k1.PrivKeySecp256k1{},
		secp256k1.PrivKeyAminoName, nil)
}

func RegisterEvidenceMessages(cdc *amino.Codec) {
	cdc.RegisterInterface((*evidence.EvidenceMessage)(nil), nil)
	cdc.RegisterConcrete(&evidence.EvidenceListMessage{},
		"tendermint/evidence/EvidenceListMessage", nil)
	cdc.RegisterInterface((*types.Evidence)(nil), nil)
	cdc.RegisterConcrete(&types.DuplicateVoteEvidence{}, "tendermint/DuplicateVoteEvidence", nil)
}
