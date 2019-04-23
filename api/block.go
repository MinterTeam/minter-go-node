package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	types2 "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

type BlockResponse struct {
	Hash         string                     `json:"hash"`
	Height       int64                      `json:"height"`
	Time         time.Time                  `json:"time"`
	NumTxs       int64                      `json:"num_txs"`
	TotalTxs     int64                      `json:"total_txs"`
	Transactions []BlockTransactionResponse `json:"transactions"`
	BlockReward  *big.Int                   `json:"block_reward"`
	Size         int                        `json:"size"`
	Proposer     types.Pubkey               `json:"proposer,omitempty"`
	Validators   []BlockValidatorResponse   `json:"validators"`
	Evidence     types2.EvidenceData        `json:"evidence,omitempty"`
}

type BlockTransactionResponse struct {
	Hash        string             `json:"hash"`
	RawTx       string             `json:"raw_tx"`
	From        string             `json:"from"`
	Nonce       uint64             `json:"nonce"`
	GasPrice    uint32             `json:"gas_price"`
	Type        transaction.TxType `json:"type"`
	Data        json.RawMessage    `json:"data"`
	Payload     []byte             `json:"payload"`
	ServiceData []byte             `json:"service_data"`
	Gas         int64              `json:"gas"`
	GasCoin     types.CoinSymbol   `json:"gas_coin"`
	Tags        map[string]string  `json:"tags"`
	Code        uint32             `json:"code,omitempty"`
	Log         string             `json:"log,omitempty"`
}

type BlockValidatorResponse struct {
	Pubkey string `json:"pub_key"`
	Signed bool   `json:"signed"`
}

func Block(height int64) (*BlockResponse, error) {
	block, err := client.Block(&height)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Block not found", Data: err.Error()}
	}

	blockResults, err := client.BlockResults(&height)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Block results not found", Data: err.Error()}
	}

	valHeight := height - 1
	if valHeight < 1 {
		valHeight = 1
	}
	tmValidators, err := client.Validators(&valHeight)
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Validators for block not found", Data: err.Error()}
	}

	txs := make([]BlockTransactionResponse, len(block.Block.Data.Txs))
	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.TxDecoder.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)
		for _, tag := range blockResults.Results.DeliverTx[i].Tags {
			tags[string(tag.Key)] = string(tag.Value)
		}

		data, err := encodeTxData(tx)
		if err != nil {
			return nil, err
		}

		txs[i] = BlockTransactionResponse{
			Hash:        fmt.Sprintf("Mt%x", rawTx.Hash()),
			RawTx:       fmt.Sprintf("%x", []byte(rawTx)),
			From:        sender.String(),
			Nonce:       tx.Nonce,
			GasPrice:    tx.GasPrice,
			Type:        tx.Type,
			Data:        data,
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         tx.Gas(),
			GasCoin:     tx.GasCoin,
			Tags:        tags,
			Code:        blockResults.Results.DeliverTx[i].Code,
			Log:         blockResults.Results.DeliverTx[i].Log,
		}
	}

	var validators []BlockValidatorResponse
	proposer := types.Pubkey{}
	if height > 1 {
		validators = make([]BlockValidatorResponse, len(tmValidators.Validators))
		for i, tmval := range tmValidators.Validators {
			signed := false
			for _, vote := range block.Block.LastCommit.Precommits {
				if vote == nil {
					continue
				}

				if bytes.Equal(vote.ValidatorAddress.Bytes(), tmval.Address.Bytes()) {
					signed = true
					break
				}
			}

			validators[i] = BlockValidatorResponse{
				Pubkey: fmt.Sprintf("Mp%x", tmval.PubKey.Bytes()[5:]),
				Signed: signed,
			}

			if bytes.Equal(tmval.Address.Bytes(), block.Block.ProposerAddress.Bytes()) {
				proposer = tmval.PubKey.Bytes()[5:]
			}
		}
	}

	return &BlockResponse{
		Hash:         hex.EncodeToString(block.Block.Hash()),
		Height:       block.Block.Height,
		Time:         block.Block.Time,
		NumTxs:       block.Block.NumTxs,
		TotalTxs:     block.Block.TotalTxs,
		Transactions: txs,
		BlockReward:  rewards.GetRewardForBlock(uint64(height)),
		Size:         len(cdc.MustMarshalBinaryLengthPrefixed(block)),
		Proposer:     proposer,
		Validators:   validators,
		Evidence:     block.Block.Evidence,
	}, nil
}
