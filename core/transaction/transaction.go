package transaction

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/crypto/sha3"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
)

const (
	TypeSend                byte = 0x01
	TypeSellCoin            byte = 0x02
	TypeSellAllCoin         byte = 0x03
	TypeBuyCoin             byte = 0x04
	TypeCreateCoin          byte = 0x05
	TypeDeclareCandidacy    byte = 0x06
	TypeDelegate            byte = 0x07
	TypeUnbond              byte = 0x08
	TypeRedeemCheck         byte = 0x09
	TypeSetCandidateOnline  byte = 0x0A
	TypeSetCandidateOffline byte = 0x0B
	TypeCreateMultisig      byte = 0x0C
)

type Transaction struct {
	Nonce       uint64
	GasPrice    *big.Int
	GasCoin     types.CoinSymbol
	Type        byte
	Data        RawData
	Payload     []byte
	ServiceData []byte
	Signatures  [][]byte

	decodedData Data
}

type RawData []byte

type Data interface {
	MarshalJSON() ([]byte, error)
	String() string
	Gas() int64
	Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response
}

func (tx *Transaction) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(tx)
}

func (tx *Transaction) Gas() int64 {
	return tx.decodedData.Gas() + tx.payloadGas()
}

func (tx *Transaction) payloadGas() int64 {
	return int64(len(tx.Payload)+len(tx.ServiceData)) * commissions.PayloadByte
}

func (tx *Transaction) String() string {
	sender, _ := tx.Sender()

	return fmt.Sprintf("TX nonce:%d from:%s payload:%s data:%s",
		tx.Nonce, sender.String(), tx.Payload, tx.decodedData.String())
}

func (tx *Transaction) Sign(prv *ecdsa.PrivateKey) error {

	h := tx.Hash()
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return err
	}

	tx.SetSignature(sig)

	return nil
}

func (tx *Transaction) SetSignature(sig []byte) {
	tx.Signatures[0] = sig
}

func (tx *Transaction) Sender() (types.Address, error) {
	return recoverPlain(tx.Hash(), tx.Signatures[0])
}

func (tx *Transaction) Hash() types.Hash {
	return rlpHash([]interface{}{
		tx.Nonce,
		tx.GasPrice,
		tx.GasCoin,
		tx.Type,
		tx.Data,
		tx.Payload,
		tx.ServiceData,
	})
}

func (tx *Transaction) SetDecodedData(data Data) {
	tx.decodedData = data
}

func (tx *Transaction) GetDecodedData() Data {
	return tx.decodedData
}

func recoverPlain(sighash types.Hash, sig []byte) (types.Address, error) {
	// recover the public key from the snature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return types.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return types.Address{}, errors.New("invalid public key")
	}
	var addr types.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}

func rlpHash(x interface{}) (h types.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func DecodeFromBytes(buf []byte) (*Transaction, error) {

	var tx Transaction
	err := rlp.Decode(bytes.NewReader(buf), &tx)

	if err != nil {
		return nil, err
	}

	switch tx.Type {
	case TypeSend:
		{
			data := SendData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.Value == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeRedeemCheck:
		{
			data := RedeemCheckData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.RawCheck == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeSellCoin:
		{
			data := SellCoinData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.ValueToSell == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeSellAllCoin:
		{
			data := SellAllCoinData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)
		}
	case TypeBuyCoin:
		{
			data := BuyCoinData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.ValueToBuy == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeCreateCoin:
		{
			data := CreateCoinData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.InitialReserve == nil || data.InitialAmount == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeDeclareCandidacy:
		{
			data := DeclareCandidacyData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.PubKey == nil || data.Stake == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeDelegate:
		{
			data := DelegateData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.PubKey == nil || data.Stake == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeSetCandidateOnline:
		{
			data := SetCandidateOnData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.PubKey == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeSetCandidateOffline:
		{
			data := SetCandidateOffData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.PubKey == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeUnbond:
		{
			data := UnbondData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.PubKey == nil || data.Value == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeCreateMultisig:
		{
			data := CreateMultisigData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)
		}
	default:
		return nil, errors.New("incorrect tx data")
	}

	if err != nil {
		return nil, err
	}

	if len(tx.Signatures) == 0 || tx.Signatures[0] == nil {
		return nil, errors.New("incorrect tx signature")
	}

	if tx.GasPrice == nil || tx.Data == nil {
		return nil, errors.New("incorrect tx data")
	}

	return &tx, nil
}
