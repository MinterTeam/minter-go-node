package transaction

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/crypto/sha3"
	"github.com/MinterTeam/minter-go-node/hexutil"
	"github.com/MinterTeam/minter-go-node/rlp"
	"math/big"
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

const (
	TypeSend                byte = 0x01
	TypeConvert             byte = 0x02
	TypeCreateCoin          byte = 0x03
	TypeDeclareCandidacy    byte = 0x04
	TypeDelegate            byte = 0x05
	TypeUnbond              byte = 0x06
	TypeRedeemCheck         byte = 0x07
	TypeSetCandidateOnline  byte = 0x08
	TypeSetCandidateOffline byte = 0x09
)

// TODO: refactor, get rid of switch cases
type Transaction struct {
	Nonce       uint64
	GasPrice    *big.Int
	Type        byte
	Data        RawData
	Payload     []byte
	ServiceData []byte
	V           *big.Int
	R           *big.Int
	S           *big.Int

	decodedData Data
}

type RawData []byte

type Data interface {
	MarshalJSON() ([]byte, error)
}

type SendData struct {
	Coin  types.CoinSymbol
	To    types.Address
	Value *big.Int
}

func (s SendData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Coin  types.CoinSymbol `json:"coin,string"`
		To    types.Address    `json:"to"`
		Value string           `json:"value"`
	}{
		Coin:  s.Coin,
		To:    s.To,
		Value: s.Value.String(),
	})
}

type SetCandidateOnData struct {
	PubKey []byte
}

func (s SetCandidateOnData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey []byte `json:"pubkey"`
	}{})
}

type SetCandidateOffData struct {
	PubKey []byte
}

func (s SetCandidateOffData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey []byte `json:"pubkey"`
	}{})
}

type ConvertData struct {
	FromCoinSymbol types.CoinSymbol
	ToCoinSymbol   types.CoinSymbol
	Value          *big.Int
}

func (s ConvertData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		FromCoin types.CoinSymbol `json:"from_coin,string"`
		ToCoin   types.CoinSymbol `json:"to_coin,string"`
		Value    string           `json:"value"`
	}{
		FromCoin: s.FromCoinSymbol,
		ToCoin:   s.ToCoinSymbol,
		Value:    s.Value.String(),
	})
}

type CreateCoinData struct {
	Name                 string
	Symbol               types.CoinSymbol
	InitialAmount        *big.Int
	InitialReserve       *big.Int
	ConstantReserveRatio uint
}

func (s CreateCoinData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name                 string           `json:"name"`
		Symbol               types.CoinSymbol `json:"coin_symbol"`
		InitialAmount        string           `json:"initial_amount"`
		InitialReserve       string           `json:"initial_reserve"`
		ConstantReserveRatio uint             `json:"constant_reserve_ratio"`
	}{
		Name:                 s.Name,
		Symbol:               s.Symbol,
		InitialAmount:        s.InitialAmount.String(),
		InitialReserve:       s.InitialReserve.String(),
		ConstantReserveRatio: s.ConstantReserveRatio,
	})
}

type DeclareCandidacyData struct {
	Address    types.Address
	PubKey     []byte
	Commission uint
	Coin       types.CoinSymbol
	Stake      *big.Int
}

func (s DeclareCandidacyData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address    types.Address
		PubKey     string
		Commission uint
		Coin       types.CoinSymbol
		Stake      string
	}{
		Address:    s.Address,
		PubKey:     fmt.Sprintf("Mp%x", s.PubKey),
		Commission: s.Commission,
		Coin:       s.Coin,
		Stake:      s.Stake.String(),
	})
}

type DelegateData struct {
	PubKey []byte
	Coin   types.CoinSymbol
	Stake  *big.Int
}

func (s DelegateData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string
		Coin   types.CoinSymbol
		Stake  string
	}{
		PubKey: fmt.Sprintf("Mp%x", s.PubKey),
		Coin:   s.Coin,
		Stake:  s.Stake.String(),
	})
}

type RedeemCheckData struct {
	RawCheck []byte
	Proof    [65]byte
}

func (s RedeemCheckData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RawCheck string
		Proof    string
	}{
		RawCheck: fmt.Sprintf("Mc%x", s.RawCheck),
		Proof:    fmt.Sprintf("%x", s.Proof),
	})
}

type UnbondData struct {
	PubKey []byte
	Coin   types.CoinSymbol
	Value  *big.Int
}

func (s UnbondData) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PubKey string
		Coin   types.CoinSymbol
		Value  string
	}{
		PubKey: fmt.Sprintf("Mp%x", s.PubKey),
		Coin:   s.Coin,
		Value:  s.Value.String(),
	})
}

func (tx *Transaction) Serialize() ([]byte, error) {

	buf, err := rlp.EncodeToBytes(tx)

	return buf, err
}

func (tx *Transaction) Gas() int64 {

	gas := int64(0)

	switch tx.Type {
	case TypeSend:
		gas = commissions.SendTx
	case TypeConvert:
		gas = commissions.ConvertTx
	case TypeCreateCoin:
		gas = commissions.CreateTx
	case TypeDeclareCandidacy:
		gas = commissions.DeclareCandidacyTx
	case TypeDelegate:
		gas = commissions.DelegateTx
	case TypeUnbond:
		gas = commissions.UnboundTx
	case TypeRedeemCheck:
		gas = commissions.RedeemCheckTx
	case TypeSetCandidateOnline:
		gas = commissions.ToggleCandidateStatus
	case TypeSetCandidateOffline:
		gas = commissions.ToggleCandidateStatus
	}

	gas = gas + int64(len(tx.Payload)+len(tx.ServiceData))*commissions.PayloadByte

	return gas
}

func (tx *Transaction) String() string {
	sender, _ := tx.Sender()

	switch tx.Type {
	case TypeSend:
		{
			txData := tx.decodedData.(SendData)
			return fmt.Sprintf("SEND TX nonce:%d from:%s to:%s coin:%s value:%s payload: %s",
				tx.Nonce, sender.String(), txData.To.String(), txData.Coin.String(), txData.Value.String(), tx.Payload)
		}
	case TypeConvert:
		{
			txData := tx.decodedData.(ConvertData)
			return fmt.Sprintf("CONVERT TX nonce:%d from:%s to:%s coin:%s value:%s payload: %s",
				tx.Nonce, sender.String(), txData.FromCoinSymbol.String(), txData.ToCoinSymbol.String(), txData.Value.String(), tx.Payload)
		}
	case TypeCreateCoin:
		{
			txData := tx.decodedData.(CreateCoinData)
			return fmt.Sprintf("CREATE COIN TX nonce:%d from:%s symbol:%s reserve:%s amount:%s crr:%d payload: %s",
				tx.Nonce, sender.String(), txData.Symbol.String(), txData.InitialReserve, txData.InitialAmount, txData.ConstantReserveRatio, tx.Payload)
		}
	case TypeDeclareCandidacy:
		{
			txData := tx.decodedData.(DeclareCandidacyData)
			return fmt.Sprintf("DECLARE CANDIDACY TX nonce:%d address:%s pubkey:%s commission: %d payload: %s",
				tx.Nonce, txData.Address.String(), hexutil.Encode(txData.PubKey[:]), txData.Commission, tx.Payload)
		}
	case TypeDelegate:
		{
			txData := tx.decodedData.(DelegateData)
			return fmt.Sprintf("DELEGATE TX nonce:%d pubkey:%s payload: %s",
				tx.Nonce, hexutil.Encode(txData.PubKey[:]), tx.Payload)
		}
	case TypeUnbond:
		{
			txData := tx.decodedData.(UnbondData)
			return fmt.Sprintf("UNBOUND TX nonce:%d pubkey:%s payload: %s",
				tx.Nonce, hexutil.Encode(txData.PubKey[:]), tx.Payload)
		}
	case TypeRedeemCheck:
		{
			txData := tx.decodedData.(RedeemCheckData)
			return fmt.Sprintf("REDEEM CHECK TX nonce:%d proof: %x",
				tx.Nonce, txData.Proof)
		}
	case TypeSetCandidateOffline:
		{
			txData := tx.decodedData.(SetCandidateOffData)
			return fmt.Sprintf("SET CANDIDATE OFFLINE TX nonce:%d, pubkey: %x",
				tx.Nonce, txData.PubKey)
		}
	case TypeSetCandidateOnline:
		{
			txData := tx.decodedData.(SetCandidateOnData)
			return fmt.Sprintf("SET CANDIDATE ONLINE TX nonce:%d, pubkey: %x",
				tx.Nonce, txData.PubKey)
		}
	}

	return "err"
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
	tx.R = new(big.Int).SetBytes(sig[:32])
	tx.S = new(big.Int).SetBytes(sig[32:64])
	tx.V = new(big.Int).SetBytes([]byte{sig[64] + 27})
}

func (tx *Transaction) Sender() (types.Address, error) {
	return recoverPlain(tx.Hash(), tx.R, tx.S, tx.V, true)
}

func (tx *Transaction) Hash() types.Hash {
	return rlpHash([]interface{}{
		tx.Nonce,
		tx.GasPrice,
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

func recoverPlain(sighash types.Hash, R, S, Vb *big.Int, homestead bool) (types.Address, error) {
	if Vb.BitLen() > 8 {
		return types.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
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

			if data.RawCheck == nil || data.Proof == nil {
				return nil, errors.New("incorrect tx data")
			}
		}
	case TypeConvert:
		{
			data := ConvertData{}
			err = rlp.Decode(bytes.NewReader(tx.Data), &data)
			tx.SetDecodedData(data)

			if data.Value == nil {
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
	default:
		return nil, errors.New("incorrect tx data")
	}

	if err != nil {
		return nil, err
	}

	if tx.S == nil || tx.R == nil || tx.V == nil {
		return nil, errors.New("incorrect tx signature")
	}

	if tx.GasPrice == nil || tx.Data == nil {
		return nil, errors.New("incorrect tx data")
	}

	return &tx, nil
}
