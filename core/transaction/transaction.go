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

	SigTypeSingle byte = 0x01
	SigTypeMulti  byte = 0x02
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

type Transaction struct {
	Nonce         uint64
	GasPrice      *big.Int
	GasCoin       types.CoinSymbol
	Type          byte
	Data          RawData
	Payload       []byte
	ServiceData   []byte
	SignatureType byte
	SignatureData []byte

	decodedData Data
	sig         *Signature
	multisig    *SignatureMulti
}

type Signature struct {
	V *big.Int
	R *big.Int
	S *big.Int
}

type SignatureMulti struct {
	Multisig   types.Address
	Signatures []Signature
}

type RawData []byte

type Data interface {
	MarshalJSON() ([]byte, error)
	String() string
	Gas() int64
	Run(sender types.Address, tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock int64) Response
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
	switch tx.SignatureType {
	case SigTypeSingle:
		{
			if tx.sig == nil {
				tx.sig = &Signature{}
			}

			tx.sig.R = new(big.Int).SetBytes(sig[:32])
			tx.sig.S = new(big.Int).SetBytes(sig[32:64])
			tx.sig.V = new(big.Int).SetBytes([]byte{sig[64] + 27})

			data, err := rlp.EncodeToBytes(tx.sig)

			if err != nil {
				panic(err)
			}

			tx.SignatureData = data
		}
	case SigTypeMulti:
		{
			if tx.multisig == nil {
				tx.multisig = &SignatureMulti{
					Multisig:   types.Address{},
					Signatures: []Signature{},
				}
			}

			tx.multisig.Signatures = append(tx.multisig.Signatures, Signature{
				V: new(big.Int).SetBytes([]byte{sig[64] + 27}),
				R: new(big.Int).SetBytes(sig[:32]),
				S: new(big.Int).SetBytes(sig[32:64]),
			})

			data, err := rlp.EncodeToBytes(tx.multisig)

			if err != nil {
				panic(err)
			}

			tx.SignatureData = data
		}
	}
}

func (tx *Transaction) Sender() (types.Address, error) {
	switch tx.SignatureType {
	case SigTypeSingle:
		return RecoverPlain(tx.Hash(), tx.sig.R, tx.sig.S, tx.sig.V)
	case SigTypeMulti:
		return tx.multisig.Multisig, nil
	default:
		return types.Address{}, errors.New("unknown signature type")
	}
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
		tx.SignatureType,
	})
}

func (tx *Transaction) SetDecodedData(data Data) {
	tx.decodedData = data
}

func (tx *Transaction) GetDecodedData() Data {
	return tx.decodedData
}

func (tx *Transaction) SetMultisigAddress(address types.Address) {
	if tx.multisig == nil {
		tx.multisig = &SignatureMulti{}
	}

	tx.multisig.Multisig = address

	data, err := rlp.EncodeToBytes(tx.multisig)

	if err != nil {
		panic(err)
	}

	tx.SignatureData = data
}

func RecoverPlain(sighash types.Hash, R, S, Vb *big.Int) (types.Address, error) {
	if Vb.BitLen() > 8 {
		return types.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S) {
		return types.Address{}, ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

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

	switch tx.SignatureType {
	case SigTypeMulti:
		{
			tx.multisig = &SignatureMulti{}
			if err := rlp.DecodeBytes(tx.SignatureData, tx.multisig); err != nil {
				return nil, err
			}
		}
	case SigTypeSingle:
		{
			tx.sig = &Signature{}
			if err := rlp.DecodeBytes(tx.SignatureData, tx.sig); err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.New("unknown signature type")
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

	if tx.GasPrice == nil || tx.Data == nil {
		return nil, errors.New("incorrect tx data")
	}

	return &tx, nil
}
