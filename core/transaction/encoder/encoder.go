package encoder

import (
	"encoding/json"
	"fmt"

	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	rpctypes "github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"github.com/tendermint/tendermint/libs/bytes"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type TxEncoderJSON struct {
	context *state.CheckState
}

type TransactionResponse struct {
	Hash     string            `json:"hash"`
	RawTx    string            `json:"raw_tx"`
	Height   int64             `json:"height"`
	Index    uint32            `json:"index"`
	From     string            `json:"from"`
	Nonce    uint64            `json:"nonce"`
	Gas      int64             `json:"gas"`
	GasPrice uint32            `json:"gas_price"`
	GasCoin  CoinResource      `json:"gas_coin"`
	Type     uint8             `json:"type"`
	Data     json.RawMessage   `json:"data"`
	Payload  []byte            `json:"payload"`
	Tags     map[string]string `json:"tags"`
	Code     uint32            `json:"code,omitempty"`
	Log      string            `json:"log,omitempty"`
}

var resourcesConfig = map[transaction.TxType]TxDataResource{
	transaction.TypeSend:                   new(SendDataResource),
	transaction.TypeSellCoin:               new(SellCoinDataResource),
	transaction.TypeSellAllCoin:            new(SellAllCoinDataResource),
	transaction.TypeBuyCoin:                new(BuyCoinDataResource),
	transaction.TypeCreateCoin:             new(CreateCoinDataResource),
	transaction.TypeDeclareCandidacy:       new(DeclareCandidacyDataResource),
	transaction.TypeDelegate:               new(DelegateDataResource),
	transaction.TypeUnbond:                 new(UnbondDataResource),
	transaction.TypeRedeemCheck:            new(RedeemCheckDataResource),
	transaction.TypeSetCandidateOnline:     new(SetCandidateOnDataResource),
	transaction.TypeSetCandidateOffline:    new(SetCandidateOffDataResource),
	transaction.TypeCreateMultisig:         new(CreateMultisigDataResource),
	transaction.TypeMultisend:              new(MultiSendDataResource),
	transaction.TypeEditCandidate:          new(EditCandidateDataResource),
	transaction.TypeSetHaltBlock:           new(SetHaltBlockDataResource),
	transaction.TypeRecreateCoin:           new(RecreateCoinDataResource),
	transaction.TypeEditCoinOwner:          new(EditCoinOwnerDataResource),
	transaction.TypeEditMultisig:           new(EditMultisigResource),
	transaction.TypePriceVote:              new(PriceVoteResource),
	transaction.TypeEditCandidatePublicKey: new(EditCandidatePublicKeyResource),
}

func NewTxEncoderJSON(context *state.CheckState) *TxEncoderJSON {
	return &TxEncoderJSON{context}
}

func (encoder *TxEncoderJSON) Encode(transaction *transaction.Transaction, tmTx *coretypes.ResultTx) (json.RawMessage, error) {
	sender, _ := transaction.Sender()

	// prepare transaction data resource
	data, err := encoder.EncodeData(transaction)
	if err != nil {
		return nil, err
	}

	// prepare transaction tags
	tags := make(map[string]string)
	for _, tag := range tmTx.TxResult.Events[0].Attributes {
		tags[string(tag.Key)] = string(tag.Value)
	}

	gasCoin := encoder.context.Coins().GetCoin(transaction.GasCoin)
	txGasCoin := CoinResource{gasCoin.ID().Uint32(), gasCoin.GetFullSymbol()}

	tx := TransactionResponse{
		Hash:     bytes.HexBytes(tmTx.Tx.Hash()).String(),
		RawTx:    fmt.Sprintf("%x", []byte(tmTx.Tx)),
		Height:   tmTx.Height,
		Index:    tmTx.Index,
		From:     sender.String(),
		Nonce:    transaction.Nonce,
		Gas:      transaction.Gas(),
		GasPrice: transaction.GasPrice,
		GasCoin:  txGasCoin,
		Type:     uint8(transaction.Type),
		Data:     data,
		Payload:  transaction.Payload,
		Tags:     tags,
		Code:     tmTx.TxResult.Code,
		Log:      tmTx.TxResult.Log,
	}

	return json.Marshal(tx)
}

func (encoder *TxEncoderJSON) EncodeData(decodedTx *transaction.Transaction) ([]byte, error) {
	if resource, exists := resourcesConfig[decodedTx.Type]; exists {
		return json.Marshal(
			resource.Transform(decodedTx.GetDecodedData(), encoder.context),
		)
	}

	return nil, rpctypes.RPCError{Code: 500, Message: "unknown tx type"}
}
