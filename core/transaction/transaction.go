package transaction

import (
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

type TxType byte
type SigType byte

const (
	TypeSend                TxType = 0x01
	TypeSellCoin            TxType = 0x02
	TypeSellAllCoin         TxType = 0x03
	TypeBuyCoin             TxType = 0x04
	TypeCreateCoin          TxType = 0x05
	TypeDeclareCandidacy    TxType = 0x06
	TypeDelegate            TxType = 0x07
	TypeUnbond              TxType = 0x08
	TypeRedeemCheck         TxType = 0x09
	TypeSetCandidateOnline  TxType = 0x0A
	TypeSetCandidateOffline TxType = 0x0B
	TypeCreateMultisig      TxType = 0x0C
	TypeMultisend           TxType = 0x0D
	TypeEditCandidate       TxType = 0x0E

	SigTypeSingle SigType = 0x01
	SigTypeMulti  SigType = 0x02
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
	MaxCoinSupply = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(15+18), nil) // 1,000,000,000,000,000 bips
)

type Transaction struct {
	Nonce         uint64
	GasPrice      *big.Int
	GasCoin       types.CoinSymbol
	Type          TxType
	Data          RawData
	Payload       []byte
	ServiceData   []byte
	SignatureType SigType
	SignatureData []byte

	decodedData Data
	sig         *Signature
	multisig    *SignatureMulti
	sender      *types.Address
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

type TotalSpends []TotalSpend

func (tss *TotalSpends) Add(coin types.CoinSymbol, value *big.Int) {
	for i, t := range *tss {
		if t.Coin == coin {
			(*tss)[i].Value.Add((*tss)[i].Value, big.NewInt(0).Set(value))
			return
		}
	}

	*tss = append(*tss, TotalSpend{
		Coin:  coin,
		Value: big.NewInt(0).Set(value),
	})
}

type TotalSpend struct {
	Coin  types.CoinSymbol
	Value *big.Int
}

type Conversion struct {
	FromCoin    types.CoinSymbol
	FromAmount  *big.Int
	FromReserve *big.Int
	ToCoin      types.CoinSymbol
	ToAmount    *big.Int
	ToReserve   *big.Int
}

type Data interface {
	String() string
	Gas() int64
	TotalSpend(tx *Transaction, context *state.StateDB) (TotalSpends, []Conversion, *big.Int, *Response)
	BasicCheck(tx *Transaction, context *state.StateDB) *Response
	Run(tx *Transaction, context *state.StateDB, isCheck bool, rewardPool *big.Int, currentBlock uint64) Response
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

func (tx *Transaction) CommissionInBaseCoin() *big.Int {
	commissionInBaseCoin := big.NewInt(0).Mul(tx.GasPrice, big.NewInt(tx.Gas()))
	commissionInBaseCoin.Mul(commissionInBaseCoin, CommissionMultiplier)

	return commissionInBaseCoin
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
	if tx.sender != nil {
		return *tx.sender, nil
	}

	switch tx.SignatureType {
	case SigTypeSingle:
		sender, err := RecoverPlain(tx.Hash(), tx.sig.R, tx.sig.S, tx.sig.V)
		if err != nil {
			return types.Address{}, err
		}

		tx.sender = &sender
		return sender, nil
	case SigTypeMulti:
		return tx.multisig.Multisig, nil
	}

	return types.Address{}, errors.New("unknown signature type")
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
	err := rlp.Encode(hw, x)
	if err != nil {
		panic(err)
	}
	hw.Sum(h[:0])
	return h
}

func CheckForCoinSupplyOverflow(current *big.Int, delta *big.Int) error {
	total := big.NewInt(0).Set(current)
	total.Add(total, delta)

	if total.Cmp(MaxCoinSupply) != -1 {
		return errors.New("сoin supply overflow")
	}

	return nil
}
