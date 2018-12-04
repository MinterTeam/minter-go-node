package api

import (
	"encoding/json"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/tendermint/tendermint/libs/common"
	tmtypes "github.com/tendermint/tendermint/types"
	"math/big"
	"time"
)

type BlockResponse struct {
	Hash         common.HexBytes            `json:"hash"`
	Height       int64                      `json:"height"`
	Time         time.Time                  `json:"time"`
	NumTxs       int64                      `json:"num_txs"`
	TotalTxs     int64                      `json:"total_txs"`
	Transactions []BlockTransactionResponse `json:"transactions"`
	Precommits   []*tmtypes.Vote            `json:"precommits"`
	BlockReward  *big.Int                   `json:"block_reward"`
	Size         int                        `json:"size"`
}

type BlockTransactionResponse struct {
	Hash        string            `json:"hash"`
	RawTx       string            `json:"raw_tx"`
	From        string            `json:"from"`
	Nonce       uint64            `json:"nonce"`
	GasPrice    *big.Int          `json:"gas_price"`
	Type        byte              `json:"type"`
	Data        json.RawMessage   `json:"data"`
	Payload     []byte            `json:"payload"`
	ServiceData []byte            `json:"service_data"`
	Gas         int64             `json:"gas"`
	GasCoin     types.CoinSymbol  `json:"gas_coin"`
	GasUsed     int64             `json:"gas_used"`
	Tags        map[string]string `json:"tags"`
	Code        uint32            `json:"code,omitempty"`
	Log         string            `json:"log,omitempty"`
}

func Block(height int64) (*BlockResponse, error) {
	block, err := client.Block(&height)
	blockResults, err := client.BlockResults(&height)
	if err != nil {
		return nil, err
	}

	txs := make([]BlockTransactionResponse, len(block.Block.Data.Txs))
	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		tags := make(map[string]string)

		for _, tag := range blockResults.Results.DeliverTx[i].Tags {
			switch string(tag.Key) {
			case "tx.type":
				tags[string(tag.Key)] = fmt.Sprintf("%X", tag.Value)
			default:
				tags[string(tag.Key)] = string(tag.Value)
			}
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
			GasUsed:     blockResults.Results.DeliverTx[i].GasUsed,
			Tags:        tags,
			Code:        blockResults.Results.DeliverTx[i].Code,
			Log:         blockResults.Results.DeliverTx[i].Log,
		}
	}

	return &BlockResponse{
		Hash:         block.Block.Hash(),
		Height:       block.Block.Height,
		Time:         block.Block.Time,
		NumTxs:       block.Block.NumTxs,
		TotalTxs:     block.Block.TotalTxs,
		Transactions: txs,
		Precommits:   block.Block.LastCommit.Precommits,
		BlockReward:  rewards.GetRewardForBlock(uint64(height)),
		Size:         len(cdc.MustMarshalBinaryLengthPrefixed(block)),
	}, nil
}
