package tests

import (
	"crypto/ecdsa"
	"time"

	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/go-amino"
	tmTypes "github.com/tendermint/tendermint/abci/types"
	tmTypes1 "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proto/tendermint/version"
)

// CreateApp creates and returns new Blockchain instance
func CreateApp(state types.AppState) *minter.Blockchain {
	jsonState, err := amino.MarshalJSON(state)
	if err != nil {
		panic(err)
	}

	storage := utils.NewStorage("", "")
	cfg := config.GetConfig(storage.GetMinterHome())
	cfg.DBBackend = "memdb"
	app := minter.NewMinterBlockchain(storage, cfg, nil, 120)
	var updates []tmTypes.ValidatorUpdate
	for _, validator := range state.Validators {
		updates = append(updates, tmTypes.Ed25519ValidatorUpdate(validator.PubKey.Bytes(), 1))
	}
	app.InitChain(tmTypes.RequestInitChain{
		Time:          time.Now(),
		ChainId:       "test",
		Validators:    updates,
		InitialHeight: 1,
		AppStateBytes: jsonState,
	})

	return app
}

// SendCommit sends Commit message to given Blockchain instance
func SendCommit(app *minter.Blockchain) tmTypes.ResponseCommit {
	return app.Commit()
}

// SendBeginBlock sends BeginBlock message to given Blockchain instance
func SendBeginBlock(app *minter.Blockchain, height int64) tmTypes.ResponseBeginBlock {
	var voteInfos []tmTypes.VoteInfo
	for _, validator := range app.CurrentState().Validators().GetValidators() {
		address := validator.GetAddress()
		voteInfos = append(voteInfos, tmTypes.VoteInfo{
			Validator: tmTypes.Validator{
				Address: address[:],
				Power:   0,
			},
			SignedLastBlock: true,
		})
	}
	return app.BeginBlock(tmTypes.RequestBeginBlock{
		Hash: nil,
		Header: tmTypes1.Header{
			Version:            version.Consensus{},
			ChainID:            "",
			Height:             height,
			Time:               time.Time{},
			LastBlockId:        tmTypes1.BlockID{},
			LastCommitHash:     nil,
			DataHash:           nil,
			ValidatorsHash:     nil,
			NextValidatorsHash: nil,
			ConsensusHash:      nil,
			AppHash:            nil,
			LastResultsHash:    nil,
			EvidenceHash:       nil,
			ProposerAddress:    nil,
		},
		LastCommitInfo: tmTypes.LastCommitInfo{
			Round: 0,
			Votes: voteInfos,
		},
		ByzantineValidators: nil,
	})
}

// SendEndBlock sends EndBlock message to given Blockchain instance
func SendEndBlock(app *minter.Blockchain, height int64) tmTypes.ResponseEndBlock {
	return app.EndBlock(tmTypes.RequestEndBlock{
		Height: height,
	})
}

// CreateTx composes and returns Tx with given params.
// Nonce, chain id, gas price, gas coin and signature type fields are auto-filled.
func CreateTx(app *minter.Blockchain, address types.Address, txType transaction.TxType, data interface{}, gas types.CoinID, gasPrice ...uint32) transaction.Transaction {
	nonce := app.CurrentState().Accounts().GetNonce(address) + 1
	bData, err := rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	var mulGas uint32 = 1
	if len(gasPrice) != 0 {
		mulGas = gasPrice[0]
	}

	tx := transaction.Transaction{
		Nonce:         nonce,
		ChainID:       types.CurrentChainID,
		GasPrice:      mulGas,
		GasCoin:       gas,
		Type:          txType,
		Data:          bData,
		SignatureType: transaction.SigTypeSingle,
	}

	return tx
}

// SendTx sends DeliverTx message to given Blockchain instance
func SendTx(app *minter.Blockchain, bytes []byte) tmTypes.ResponseDeliverTx {
	return app.DeliverTx(tmTypes.RequestDeliverTx{
		Tx: bytes,
	})
}

// SignTx returns bytes of signed with given pk transaction
func SignTx(pk *ecdsa.PrivateKey, tx transaction.Transaction) []byte {
	err := tx.Sign(pk)
	if err != nil {
		panic(err)
	}

	b, _ := rlp.EncodeToBytes(tx)

	return b
}

// CreateAddress returns random address and corresponding private key
func CreateAddress() (types.Address, *ecdsa.PrivateKey) {
	pk, _ := crypto.GenerateKey()

	return crypto.PubkeyToAddress(pk.PublicKey), pk
}

// DefaultAppState returns new AppState with some predefined values
func DefaultAppState() types.AppState {
	return types.AppState{
		Note:                "",
		Validators:          nil,
		Candidates:          nil,
		BlockListCandidates: nil,
		Waitlist:            nil,
		Pools:               nil,
		Accounts:            nil,
		Coins:               nil,
		FrozenFunds:         nil,
		HaltBlocks:          nil,
		Commission: types.Commission{
			Coin:                    0,
			PayloadByte:             "2000000000000000",
			Send:                    "10000000000000000",
			BuyBancor:               "100000000000000000",
			SellBancor:              "100000000000000000",
			SellAllBancor:           "100000000000000000",
			BuyPoolBase:             "100000000000000000",
			BuyPoolDelta:            "50000000000000000",
			SellPoolBase:            "100000000000000000",
			SellPoolDelta:           "50000000000000000",
			SellAllPoolBase:         "100000000000000000",
			SellAllPoolDelta:        "50000000000000000",
			CreateTicker3:           "1000000000000000000000000",
			CreateTicker4:           "100000000000000000000000",
			CreateTicker5:           "10000000000000000000000",
			CreateTicker6:           "1000000000000000000000",
			CreateTicker7_10:        "100000000000000000000",
			CreateCoin:              "0",
			CreateToken:             "0",
			RecreateCoin:            "10000000000000000000000",
			RecreateToken:           "10000000000000000000000",
			DeclareCandidacy:        "10000000000000000000",
			Delegate:                "200000000000000000",
			Unbond:                  "200000000000000000",
			RedeemCheck:             "30000000000000000",
			SetCandidateOn:          "100000000000000000",
			SetCandidateOff:         "100000000000000000",
			CreateMultisig:          "100000000000000000",
			MultisendBase:           "10000000000000000",
			MultisendDelta:          "5000000000000000",
			EditCandidate:           "10000000000000000000",
			SetHaltBlock:            "1000000000000000000",
			EditTickerOwner:         "10000000000000000000000",
			EditMultisig:            "1000000000000000000",
			EditCandidatePublicKey:  "100000000000000000000000",
			CreateSwapPool:          "1000000000000000000",
			AddLiquidity:            "100000000000000000",
			RemoveLiquidity:         "100000000000000000",
			EditCandidateCommission: "10000000000000000000",
			MintToken:               "100000000000000000",
			BurnToken:               "100000000000000000",
			VoteCommission:          "1000000000000000000",
			VoteUpdate:              "1000000000000000000",
			FailedTx:                "",
			AddLimitOrder:           "",
			RemoveLimitOrder:        "",
		},
		CommissionVotes: nil,
		UpdateVotes:     nil,
		UsedChecks:      nil,
		MaxGas:          0,
		TotalSlashed:    "0",
	}
}
