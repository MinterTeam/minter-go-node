package transaction

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
	"testing"
)

func TestDecodeFromBytesToInvalidSignature(t *testing.T) {
	data := SendData{Coin: 0, To: types.Address{0}, Value: big.NewInt(0)}
	encodedData, err := rlp.EncodeToBytes(data)
	if err != nil {
		t.Fatal(err)
	}

	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSend,
		Data:          encodedData,
		SignatureType: SigTypeSingle,
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = TxDecoder.DecodeFromBytes(encodedTx)
	if err == nil {
		t.Fatal("Expected the invalid signature error")
	}
}

func TestDecodeSigToInvalidMultiSignature(t *testing.T) {
	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSend,
		Data:          nil,
		SignatureType: SigTypeMulti,
	}

	_, err := DecodeSig(&tx)
	if err == nil {
		t.Fatal("Expected the invalid signature error")
	}
}

func TestDecodeSigToInvalidSingleSignature(t *testing.T) {
	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSend,
		Data:          nil,
		SignatureType: SigTypeSingle,
	}

	_, err := DecodeSig(&tx)
	if err == nil {
		t.Fatal("Expected the invalid signature error")
	}
}

func TestDecodeSigToUnknownSignatureType(t *testing.T) {
	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          TypeSend,
		Data:          nil,
		SignatureType: 0x03,
	}

	_, err := DecodeSig(&tx)
	if err == nil {
		t.Fatal("Expected unknown signature type error")
	}
}

func TestDecodeFromBytesWithoutSigToInvalidData(t *testing.T) {
	tx := Transaction{
		Nonce:         1,
		GasPrice:      1,
		ChainID:       types.CurrentChainID,
		GasCoin:       types.GetBaseCoinID(),
		Type:          0x20,
		Data:          nil,
		SignatureType: SigTypeSingle,
	}

	encodedTx, err := rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = TxDecoder.DecodeFromBytesWithoutSig(encodedTx)
	if err == nil {
		t.Fatal("Expected tx type is not registered error")
	}

	tx.Type = TypeSend
	encodedTx, err = rlp.EncodeToBytes(tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = TxDecoder.DecodeFromBytesWithoutSig(encodedTx)
	if err == nil {
		t.Fatal("Expected invalid data error")
	}
}
