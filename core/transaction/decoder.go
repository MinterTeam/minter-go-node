package transaction

import (
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/rlp"
	"reflect"
)

var TxDecoder = Decoder{
	registeredTypes: map[TxType]Data{},
}

func init() {
	TxDecoder.RegisterType(TypeSend, SendData{})
	TxDecoder.RegisterType(TypeSellCoin, SellCoinData{})
	TxDecoder.RegisterType(TypeSellAllCoin, SellAllCoinData{})
	TxDecoder.RegisterType(TypeBuyCoin, BuyCoinData{})
	TxDecoder.RegisterType(TypeCreateCoin, CreateCoinData{})
	TxDecoder.RegisterType(TypeDeclareCandidacy, DeclareCandidacyData{})
	TxDecoder.RegisterType(TypeDelegate, DelegateData{})
	TxDecoder.RegisterType(TypeUnbond, UnbondData{})
	TxDecoder.RegisterType(TypeRedeemCheck, RedeemCheckData{})
	TxDecoder.RegisterType(TypeSetCandidateOnline, SetCandidateOnData{})
	TxDecoder.RegisterType(TypeSetCandidateOffline, SetCandidateOffData{})
	TxDecoder.RegisterType(TypeCreateMultisig, CreateMultisigData{})
	TxDecoder.RegisterType(TypeMultisend, MultisendData{})
	TxDecoder.RegisterType(TypeEditCandidate, EditCandidateData{})
}

type Decoder struct {
	registeredTypes map[TxType]Data
}

func (decoder *Decoder) RegisterType(t TxType, d Data) {
	decoder.registeredTypes[t] = d
}

func (decoder *Decoder) DecodeFromBytes(buf []byte) (*Transaction, error) {
	var tx Transaction
	err := rlp.DecodeBytes(buf, &tx)

	if err != nil {
		return nil, err
	}

	if tx.GasPrice == nil || tx.Data == nil {
		return nil, errors.New("incorrect tx data")
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

	d, ok := decoder.registeredTypes[tx.Type]

	if !ok {
		return nil, errors.New(fmt.Sprintf("tx type %x is not registered", tx.Type))
	}

	err = rlp.DecodeBytesForType(tx.Data, reflect.ValueOf(d).Type(), &d)

	if err != nil {
		return nil, err
	}

	tx.SetDecodedData(d)

	return &tx, nil
}
