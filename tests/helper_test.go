package tests

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/state/coins"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"sort"
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

func init() {
	types.CurrentChainID = types.ChainTestnet
}

func CreateAppDefault(state types.AppState) *minter.Blockchain {
	const (
		updateStakePeriod   = 12
		expiredOrdersPeriod = 24
	)
	return CreateApp(state, updateStakePeriod, expiredOrdersPeriod, 999)
}

func CreateApp(state types.AppState, updateStakePeriod, expiredOrdersPeriod uint64, initialHeightOmitempty uint64) *minter.Blockchain {

	var lastUpdateHeight uint64
	var votes []types.UpdateVote
	for i, vote := range state.UpdateVotes {
		lastUpdateHeight = initialHeightOmitempty + uint64(i) + 2
		votes = append(votes, types.UpdateVote{
			Height:  lastUpdateHeight,
			Votes:   vote.Votes,
			Version: vote.Version,
		})
	}
	state.UpdateVotes = votes

	jsonState, err := amino.MarshalJSON(state)
	if err != nil {
		panic(err)
	}

	storage := utils.NewStorage("", "")
	cfg := config.GetConfig(storage.GetMinterHome())
	cfg.DBBackend = "memdb"

	app := minter.NewMinterBlockchain(storage, cfg, nil, updateStakePeriod, expiredOrdersPeriod, nil)
	var updates []tmTypes.ValidatorUpdate
	for _, validator := range state.Validators {
		updates = append(updates, tmTypes.Ed25519ValidatorUpdate(validator.PubKey.Bytes(), 1))
	}
	app.InitChain(tmTypes.RequestInitChain{
		Time:          time.Unix(0, 0),
		ChainId:       "test1",
		Validators:    updates,
		InitialHeight: int64(initialHeightOmitempty),
		AppStateBytes: jsonState,
	})

	for i := initialHeightOmitempty; i < lastUpdateHeight+1; i++ {
		SendBeginBlock(app, time.Unix(int64(lastUpdateHeight), 0))
		SendEndBlock(app)
		SendCommit(app)
	}

	return app
}

type Helper struct {
	app           *minter.Blockchain
	lastBlockTime time.Time
}

func NewHelper(state types.AppState) *Helper {
	return &Helper{lastBlockTime: time.Unix(0, 0), app: CreateAppDefault(state)}
}

func (h *Helper) NextBlock(txs ...transaction.Transaction) (height uint64, results []tmTypes.ResponseDeliverTx) {
	h.lastBlockTime = h.lastBlockTime.Add(time.Second)
	SendBeginBlock(h.app, h.lastBlockTime)
	for _, tx := range txs {
		b, err := rlp.EncodeToBytes(tx)
		if err != nil {
			panic(err)
		}
		results = append(results, SendTx(h.app, b))
	}
	SendEndBlock(h.app)
	return SendCommit(h.app), results
}

func SendCommit(app *minter.Blockchain) (height uint64) {
	app.Commit()
	return app.Height()
}

func SendBeginBlock(app *minter.Blockchain, t time.Time) tmTypes.ResponseBeginBlock {
	var voteInfos []tmTypes.VoteInfo
	validators := app.CurrentState().Validators().GetValidators()
	for _, validator := range validators {
		address := validator.GetAddress()
		voteInfos = append(voteInfos, tmTypes.VoteInfo{
			Validator: tmTypes.Validator{
				Address: address[:],
				Power:   int64(100 / len(validators)),
			},
			SignedLastBlock: true,
		})
	}

	return app.BeginBlock(tmTypes.RequestBeginBlock{
		Hash: nil,
		Header: tmTypes1.Header{
			Version:            version.Consensus{},
			ChainID:            "test1",
			Height:             int64(app.Height() + 1),
			Time:               t,
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

func SendEndBlock(app *minter.Blockchain) tmTypes.ResponseEndBlock {
	return app.EndBlock(tmTypes.RequestEndBlock{
		Height: int64(app.Height() + 1),
	})
}

// CreateTx composes and returns Tx with given params.
// Nonce, chain id, gas price, gas coin and signature type fields are auto-filled.
func (h *Helper) CreateTx(pk *ecdsa.PrivateKey, data transaction.Data, gas types.CoinID, gasPrice ...uint32) transaction.Transaction {
	address := crypto.PubkeyToAddress(pk.PublicKey)

	nonce := h.app.CurrentState().Accounts().GetNonce(address) + 1
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
		Type:          data.TxType(),
		Data:          bData,
		SignatureType: transaction.SigTypeSingle,
	}

	err = tx.Sign(pk)
	if err != nil {
		panic(err)
	}

	d, ok := transaction.GetData(data.TxType())
	if !ok {
		panic(fmt.Sprintf("tx type %x is not registered", tx.Type))
	}

	err = rlp.DecodeBytes(tx.Data, d)

	if err != nil {
		panic(err)
	}

	tx.SetDecodedData(d)

	return tx
}

// SendTx sends DeliverTx message to given Blockchain instance
func SendTx(app *minter.Blockchain, bytes []byte) tmTypes.ResponseDeliverTx {
	return app.DeliverTx(tmTypes.RequestDeliverTx{
		Tx: bytes,
	})
}

type User struct {
	address    types.Address
	privateKey *ecdsa.PrivateKey
}

// CreateAddress returns random address and corresponding private key
func CreateAddress() *User {
	pk, _ := crypto.GenerateKey()

	return &User{crypto.PubkeyToAddress(pk.PublicKey), pk}
}

var initialBIPStake = helpers.StringToBigInt("1000000000000000000000000000000000")

// DefaultAppState returns new AppState with some predefined values
func DefaultAppState(addresses ...types.Address) types.AppState {
	var accounts = make([]types.Account, 0, len(addresses))
	var validators = make([]types.Validator, 0, len(addresses))
	var candidates = make([]types.Candidate, 0, len(addresses))

	for i, addr := range addresses {
		accounts = append(accounts, types.Account{
			Address: addr,
			Balance: []types.Balance{{
				Coin:  0,
				Value: "100000000000000000000000000",
			}, {
				Coin:  1,
				Value: "100000000000000000000000",
			}, {
				Coin:  types.USDTID,
				Value: "100000000000000000000000",
			}},
			Nonce:               0,
			MultisigData:        nil,
			LockStakeUntilBlock: 0,
		})
		validators = append(validators, types.Validator{
			TotalBipStake: "200000000000000000000000000",
			PubKey:        getValidatorAddress(i),
			AccumReward:   "0",
			AbsentTimes:   types.NewBitArray(24),
		})
		candidates = append(candidates, types.Candidate{
			ID:             uint64(i + 1),
			RewardAddress:  getRewardAddress(i),
			OwnerAddress:   addr,
			ControlAddress: addr,
			TotalBipStake:  "200000000000000000000000000",
			PubKey:         getValidatorAddress(i),
			Commission:     10,
			Stakes: []types.Stake{{
				Owner:    addr,
				Coin:     0,
				Value:    initialBIPStake.String(),
				BipValue: initialBIPStake.String(),
			}, {
				Owner:    getCustomAddress(i),
				Coin:     0,
				Value:    "100000000000000000000000000",
				BipValue: "100000000000000000000000000",
			}},
			//Updates: nil,
			Updates: []types.Stake{{
				Owner:    addr,
				Coin:     0,
				Value:    "50000000000000000000000000",
				BipValue: "50000000000000000000000000",
			}},
			Status:                   2,
			JailedUntil:              0,
			LastEditCommissionHeight: 0,
		})
	}

	var votes []types.UpdateVote
	for _, v := range []string{minter.V310, minter.V320, minter.V330, minter.V340} {
		vote := types.UpdateVote{
			Height:  0,
			Votes:   nil,
			Version: v,
		}
		for i := range addresses {
			vote.Votes = append(vote.Votes, getValidatorAddress(i))
		}
		votes = append(votes, vote)
	}

	return types.AppState{
		Note:                "test1",
		Validators:          validators,
		Candidates:          candidates,
		BlockListCandidates: nil,
		DeletedCandidates:   nil,
		Waitlist:            nil,
		Pools: []types.Pool{{
			Coin0:    0,
			Coin1:    types.USDTID,
			Reserve0: "100000000000000000000000000",
			Reserve1: "100000000000000000000000",
			ID:       1,
			Orders:   nil,
		}},
		NextOrderID: 0,
		Accounts:    accounts,
		Coins: []types.Coin{
			{
				ID:           1,
				Name:         "Reserve Coin 1",
				Symbol:       types.StrToCoinSymbol("COIN1RES"),
				Volume:       fmt.Sprintf("%d00000000000000000000000", len(accounts)),
				Crr:          50,
				Reserve:      "100000000000000000000",
				MaxSupply:    coins.MaxCoinSupply().String(),
				Version:      0,
				OwnerAddress: &types.Address{},
				Mintable:     false,
				Burnable:     false,
			},
			{
				ID:           types.USDTID,
				Name:         "USDT Eth",
				Symbol:       types.StrToCoinSymbol("USDTE"),
				Volume:       fmt.Sprintf("%d00000000000000000000000", len(accounts)+1),
				Crr:          0,
				Reserve:      "0",
				MaxSupply:    coins.MaxCoinSupply().String(),
				Version:      0,
				OwnerAddress: nil,
				Mintable:     true,
				Burnable:     true,
			},
		},
		FrozenFunds: nil,
		HaltBlocks:  nil,
		Commission: types.Commission{
			Coin:                    types.USDTID,
			PayloadByte:             "2000000000000",
			Send:                    "10000000000000",
			BuyBancor:               "100000000000000",
			SellBancor:              "100000000000000",
			SellAllBancor:           "100000000000000",
			BuyPoolBase:             "100000000000000",
			BuyPoolDelta:            "50000000000000",
			SellPoolBase:            "100000000000000",
			SellPoolDelta:           "50000000000000",
			SellAllPoolBase:         "100000000000000",
			SellAllPoolDelta:        "50000000000000",
			CreateTicker3:           "1000000000000000000000",
			CreateTicker4:           "100000000000000000000",
			CreateTicker5:           "10000000000000000000",
			CreateTicker6:           "1000000000000000000",
			CreateTicker7_10:        "100000000000000000",
			CreateCoin:              "200000000000000",
			CreateToken:             "200000000000000",
			RecreateCoin:            "10000000000000000000",
			RecreateToken:           "10000000000000000000",
			DeclareCandidacy:        "10000000000000000",
			Delegate:                "200000000000000",
			Unbond:                  "200000000000000",
			RedeemCheck:             "30000000000000",
			SetCandidateOn:          "100000000000000",
			SetCandidateOff:         "100000000000000",
			CreateMultisig:          "100000000000000",
			MultisendBase:           "10000000000000",
			MultisendDelta:          "5000000000000",
			EditCandidate:           "10000000000000000",
			SetHaltBlock:            "1000000000000000",
			EditTickerOwner:         "10000000000000000000",
			EditMultisig:            "1000000000000000",
			EditCandidatePublicKey:  "100000000000000000000",
			CreateSwapPool:          "1000000000000000",
			AddLiquidity:            "100000000000000",
			RemoveLiquidity:         "100000000000000",
			EditCandidateCommission: "10000000000000000",
			MintToken:               "100000000000000",
			BurnToken:               "100000000000000",
			VoteCommission:          "1000000000000000",
			VoteUpdate:              "1000000000000000",
			FailedTx:                "5000000000000",
			AddLimitOrder:           "300000000000000",
			RemoveLimitOrder:        "100000000000000",
			MoveStake:               "500000000000000",
			LockStake:               "100000000000000",
			Lock:                    "200000000000000",
		},
		CommissionVotes: nil,
		UpdateVotes:     votes,
		UsedChecks:      nil,
		MaxGas:          0,
		TotalSlashed:    "0",
		Emission:        "9999",
		PrevReward: types.RewardPrice{
			Time:       0,
			AmountBIP:  "350",
			AmountUSDT: "1",
			Off:        false,
			Reward:     "74000000000000000000",
		},
		Version:  "v300",
		Versions: nil,
	}
}

func getValidatorAddress(i int) types.Pubkey {
	return types.Pubkey{byte(i)}
}

func getRewardAddress(i int) (addr types.Address) {
	copy(addr[:], big.NewInt(int64(i)).Bytes())
	return
}
func getCustomAddress(i int) (addr types.Address) {
	bytes := big.NewInt(int64(i)).Bytes()
	copy(addr[:], bytes)
	sort.Slice(addr[:], func(i, j int) bool {
		return false
	})
	return
}
