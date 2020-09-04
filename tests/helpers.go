package tests

import (
	"crypto/ecdsa"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/MinterTeam/minter-go-node/core/minter"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"github.com/tendermint/go-amino"
	tmTypes "github.com/tendermint/tendermint/abci/types"
	"os"
	"path/filepath"
	"time"
)

func CreateApp(state types.AppState) *minter.Blockchain {
	utils.MinterHome = os.ExpandEnv(filepath.Join("$HOME", ".minter_test"))
	_ = os.RemoveAll(utils.MinterHome)

	jsonState, err := amino.MarshalJSON(state)
	if err != nil {
		panic(err)
	}

	cfg := config.GetConfig()
	app := minter.NewMinterBlockchain(cfg)
	app.InitChain(tmTypes.RequestInitChain{
		Time:    time.Now(),
		ChainId: "test",
		Validators: []tmTypes.ValidatorUpdate{
			{
				PubKey: tmTypes.PubKey{},
				Power:  1,
			},
		},
		AppStateBytes: jsonState,
	})

	return app
}

func SendCommit(app *minter.Blockchain) tmTypes.ResponseCommit {
	return app.Commit()
}

func SendBeginBlock(app *minter.Blockchain) tmTypes.ResponseBeginBlock {
	return app.BeginBlock(tmTypes.RequestBeginBlock{
		Hash: nil,
		Header: tmTypes.Header{
			Version:            tmTypes.Version{},
			ChainID:            "",
			Height:             1,
			Time:               time.Time{},
			LastBlockId:        tmTypes.BlockID{},
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
			Votes: nil,
		},
		ByzantineValidators: nil,
	})
}

func SendEndBlock(app *minter.Blockchain) tmTypes.ResponseEndBlock {
	return app.EndBlock(tmTypes.RequestEndBlock{
		Height: 0,
	})
}

func CreateTx(app *minter.Blockchain, address types.Address, txType transaction.TxType, data interface{}) transaction.Transaction {
	nonce := app.CurrentState().Accounts().GetNonce(address) + 1
	bData, err := rlp.EncodeToBytes(data)
	if err != nil {
		panic(err)
	}

	tx := transaction.Transaction{
		Nonce:         nonce,
		ChainID:       types.CurrentChainID,
		GasPrice:      1,
		GasCoin:       types.GetBaseCoinID(),
		Type:          txType,
		Data:          bData,
		SignatureType: transaction.SigTypeSingle,
	}

	return tx
}

func SendTx(app *minter.Blockchain, bytes []byte) tmTypes.ResponseDeliverTx {
	return app.DeliverTx(tmTypes.RequestDeliverTx{
		Tx: bytes,
	})
}

func SignTx(pk *ecdsa.PrivateKey, tx transaction.Transaction) []byte {
	err := tx.Sign(pk)
	if err != nil {
		panic(err)
	}

	b, _ := rlp.EncodeToBytes(tx)

	return b
}

func CreateAddress() (types.Address, *ecdsa.PrivateKey) {
	pk, _ := crypto.GenerateKey()

	return crypto.PubkeyToAddress(pk.PublicKey), pk
}

func DefaultAppState() types.AppState {
	return types.AppState{
		Note:                "",
		StartHeight:         1,
		Validators:          nil,
		Candidates:          nil,
		BlockListCandidates: nil,
		Accounts:            nil,
		Coins:               nil,
		FrozenFunds:         nil,
		HaltBlocks:          nil,
		UsedChecks:          nil,
		MaxGas:              0,
		TotalSlashed:        "0",
	}
}
