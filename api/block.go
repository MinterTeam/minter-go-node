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
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes "github.com/tendermint/tendermint/types"
	tm_types "github.com/tendermint/tendermint/types"
	"time"
)

type BlockResponse struct {
	Hash         string                     `json:"hash"`
	Height       int64                      `json:"height"`
	Time         time.Time                  `json:"time"`
	NumTxs       int64                      `json:"num_txs"`
	TotalTxs     int64                      `json:"total_txs"`
	Transactions []BlockTransactionResponse `json:"transactions"`
	BlockReward  string                     `json:"block_reward"`
	Size         int                        `json:"size"`
	Proposer     *string                    `json:"proposer,omitempty"`
	Validators   []BlockValidatorResponse   `json:"validators,omitempty"`
	Evidence     tmTypes.EvidenceData       `json:"evidence,omitempty"`
}

type BlockTransactionResponse struct {
	Hash        string            `json:"hash"`
	RawTx       string            `json:"raw_tx"`
	From        string            `json:"from"`
	Nonce       uint64            `json:"nonce"`
	GasPrice    uint32            `json:"gas_price"`
	Type        uint8             `json:"type"`
	Data        json.RawMessage   `json:"data"`
	Payload     []byte            `json:"payload"`
	ServiceData []byte            `json:"service_data"`
	Gas         int64             `json:"gas"`
	GasCoin     string            `json:"gas_coin"`
	Tags        map[string]string `json:"tags"`
	Code        uint32            `json:"code,omitempty"`
	Log         string            `json:"log,omitempty"`
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
	var allValidators []*tm_types.Validator
	for i := 1; ; i++ {
		tmValidators, err := client.Validators(&valHeight, i, 256)
		if err != nil {
			return nil, rpctypes.RPCError{Code: 404, Message: "Validators for block not found", Data: err.Error()}
		}
		if len(tmValidators.Validators) == 0 {
			break
		}
		allValidators = append(allValidators, tmValidators.Validators...)
	}
	if err != nil {
		return nil, rpctypes.RPCError{Code: 404, Message: "Validators for block not found", Data: err.Error()}
	}

	txs := make([]BlockTransactionResponse, len(block.Block.Data.Txs))
	for i, rawTx := range block.Block.Data.Txs {
		tx, _ := transaction.TxDecoder.DecodeFromBytes(rawTx)
		sender, _ := tx.Sender()

		if len(blockResults.TxsResults) == 0 {
			break
		}

		tags := make(map[string]string)
		for _, tag := range blockResults.TxsResults[i].Events[0].Attributes {
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
			Type:        uint8(tx.Type),
			Data:        data,
			Payload:     tx.Payload,
			ServiceData: tx.ServiceData,
			Gas:         tx.Gas(),
			GasCoin:     tx.GasCoin.String(),
			Tags:        tags,
			Code:        blockResults.TxsResults[i].Code,
			Log:         blockResults.TxsResults[i].Log,
		}
	}

	var validators []BlockValidatorResponse
	var proposer *string
	if height > 1 {
		p, err := getBlockProposer(block)
		if err != nil {
			return nil, err
		}

		if p != nil {
			str := p.String()
			proposer = &str
		}

		validators = make([]BlockValidatorResponse, len(allValidators))
		for i, tmval := range allValidators {
			signed := false
			for _, vote := range block.Block.LastCommit.Signatures {
				if bytes.Equal(vote.ValidatorAddress.Bytes(), tmval.Address.Bytes()) {
					signed = true
					break
				}
			}

			validators[i] = BlockValidatorResponse{
				Pubkey: fmt.Sprintf("Mp%x", tmval.PubKey.Bytes()[5:]),
				Signed: signed,
			}
		}
	}

	return &BlockResponse{
		Hash:         hex.EncodeToString(block.Block.Hash()),
		Height:       block.Block.Height,
		Time:         block.Block.Time,
		TotalTxs:     int64(len(block.Block.Txs)),
		Transactions: txs,
		BlockReward:  rewards.GetRewardForBlock(uint64(height)).String(),
		Size:         len(cdc.MustMarshalBinaryLengthPrefixed(block)),
		Proposer:     proposer,
		Validators:   validators,
		Evidence:     block.Block.Evidence,
	}, nil
}

func getBlockProposer(block *core_types.ResultBlock) (*types.Pubkey, error) {
	var allValidators []*tm_types.Validator
	for i := 1; ; i++ {
		tmValidators, err := client.Validators(&block.Block.Height, i, 256)
		if err != nil {
			return nil, err
		}
		if len(tmValidators.Validators) == 0 {
			break
		}
		allValidators = append(allValidators, tmValidators.Validators...)
	}

	for _, tmval := range allValidators {
		if bytes.Equal(tmval.Address.Bytes(), block.Block.ProposerAddress.Bytes()) {
			var result types.Pubkey
			copy(result[:], tmval.PubKey.Bytes()[5:])
			return &result, nil
		}
	}

	return nil, rpctypes.RPCError{Code: 404, Message: "Block proposer not found"}
}
