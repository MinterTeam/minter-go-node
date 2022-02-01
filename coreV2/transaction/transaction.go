package transaction

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/state"
	"github.com/MinterTeam/minter-go-node/coreV2/state/commission"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/MinterTeam/minter-go-node/crypto"
	"github.com/MinterTeam/minter-go-node/rlp"
	"golang.org/x/crypto/sha3"
)

// TxType of transaction is determined by a single byte.
type TxType byte

func (t TxType) String() string {
	return "0x" + hex.EncodeToString([]byte{byte(t)})
}

func (t TxType) UInt64() uint64 {
	return uint64(t)
}

const (
	TypeSend                    TxType = 0x01
	TypeSellCoin                TxType = 0x02
	TypeSellAllCoin             TxType = 0x03
	TypeBuyCoin                 TxType = 0x04
	TypeCreateCoin              TxType = 0x05
	TypeDeclareCandidacy        TxType = 0x06
	TypeDelegate                TxType = 0x07
	TypeUnbond                  TxType = 0x08
	TypeRedeemCheck             TxType = 0x09
	TypeSetCandidateOnline      TxType = 0x0A
	TypeSetCandidateOffline     TxType = 0x0B
	TypeCreateMultisig          TxType = 0x0C
	TypeMultisend               TxType = 0x0D
	TypeEditCandidate           TxType = 0x0E
	TypeSetHaltBlock            TxType = 0x0F
	TypeRecreateCoin            TxType = 0x10
	TypeEditCoinOwner           TxType = 0x11
	TypeEditMultisig            TxType = 0x12
	TypePriceVote               TxType = 0x13
	TypeEditCandidatePublicKey  TxType = 0x14
	TypeAddLiquidity            TxType = 0x15
	TypeRemoveLiquidity         TxType = 0x16
	TypeSellSwapPool            TxType = 0x17
	TypeBuySwapPool             TxType = 0x18
	TypeSellAllSwapPool         TxType = 0x19
	TypeEditCandidateCommission TxType = 0x1A
	TypeMoveStake               TxType = 0x1B
	TypeMintToken               TxType = 0x1C
	TypeBurnToken               TxType = 0x1D
	TypeCreateToken             TxType = 0x1E
	TypeRecreateToken           TxType = 0x1F
	TypeVoteCommission          TxType = 0x20
	TypeVoteUpdate              TxType = 0x21
	TypeCreateSwapPool          TxType = 0x22
	TypeAddLimitOrder           TxType = 0x23
	TypeRemoveLimitOrder        TxType = 0x24
	TypeLockStake               TxType = 0x25
)

const (
	gasBase           = 15
	gasSign           = 20
	gasSend           = 1
	gasMultisendBase  = 1
	gasMultisendDelta = 1

	gasCreateSwapPool  = 10
	gasAddLiquidity    = 5
	gasRemoveLiquidity = 5

	gasAddLimitOrder    = 50
	gasRemoveLimitOrder = 50

	convertDelta       = 1
	gasSellSwapPool    = 2
	gasBuySwapPool     = 2
	gasSellAllSwapPool = 2
	gasSellCoin        = 2
	gasSellAllCoin     = 2
	gasBuyCoin         = 2

	gasCreateCoin    = 3
	gasRecreateCoin  = 5
	gasCreateToken   = 3
	gasRecreateToken = 5
	gasEditCoinOwner = 5

	gasMintToken = 1
	gasBurnToken = 1

	gasRedeemCheck = 20

	gasDeclareCandidacy = 10
	gasDelegate         = 6
	gasUnbond           = 6
	gasMoveStake        = 6
	gasLockStake        = 2

	gasSetCandidateOnline      = 1
	gasSetCandidateOffline     = 1
	gasEditCandidate           = 5
	gasEditCandidatePublicKey  = 10
	gasEditCandidateCommission = 1

	gasCreateMultisig = 20
	gasEditMultisig   = 5

	gasSetHaltBlock   = 5
	gasVoteCommission = 5
	gasVoteUpdate     = 5
)

type SigType byte

const (
	SigTypeSingle SigType = 0x01
	SigTypeMulti  SigType = 0x02
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

type Transaction struct {
	Nonce         uint64
	ChainID       types.ChainID
	GasPrice      uint32
	GasCoin       types.CoinID
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

type totalSpends []totalSpend

func (tss *totalSpends) Add(coin types.CoinID, value *big.Int) {
	for i, t := range *tss {
		if t.Coin == coin {
			(*tss)[i].Value.Add((*tss)[i].Value, big.NewInt(0).Set(value))
			return
		}
	}

	*tss = append(*tss, totalSpend{
		Coin:  coin,
		Value: big.NewInt(0).Set(value),
	})
}

type totalSpend struct {
	Coin  types.CoinID
	Value *big.Int
}

type conversion struct {
	FromCoin    types.CoinID
	FromAmount  *big.Int
	FromReserve *big.Int
	ToCoin      types.CoinID
	ToAmount    *big.Int
	ToReserve   *big.Int
}

type Data interface {
	String() string
	CommissionData(*commission.Price) *big.Int
	Run(tx *Transaction, context state.Interface, rewardPool *big.Int, currentBlock uint64, price *big.Int) Response
	TxType() TxType
	Gas() int64
}

func (tx *Transaction) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(tx)
}

func (tx *Transaction) Gas() int64 {
	base := int64(gasBase)
	if tx.CommissionCoin() != types.GetBaseCoinID() {
		base += 1
	}
	if tx.payloadAndServiceDataLen() != 0 {
		base += tx.payloadAndServiceDataLen() / 1000
	}
	if tx.SignatureType == SigTypeMulti {
		base += int64(len(tx.multisig.Signatures)) * gasSign
	}
	return base + tx.decodedData.Gas()
}

func (tx *Transaction) Price(price *commission.Price) *big.Int {
	payloadAndServiceData := big.NewInt(0).Mul(big.NewInt(tx.payloadAndServiceDataLen()), price.PayloadByte)
	return big.NewInt(0).Add(tx.decodedData.CommissionData(price), payloadAndServiceData)
}

func (tx *Transaction) payloadAndServiceDataLen() int64 {
	return int64(len(tx.Payload) + len(tx.ServiceData))
}

// MulGasPrice returns price multiplier gasPrice
func (tx *Transaction) MulGasPrice(price *big.Int) *big.Int {
	return big.NewInt(0).Mul(big.NewInt(int64(tx.GasPrice)), price)
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

func (tx *Transaction) MustSender() types.Address {
	sender, err := tx.Sender()
	if err != nil {
		panic(err)
	}
	return sender
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
		tx.ChainID,
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
	if !crypto.ValidateSignatureValues(V, R, S, true) {
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
	hw := sha3.NewLegacyKeccak256()
	err := rlp.Encode(hw, x)
	if err != nil {
		panic(err)
	}
	hw.Sum(h[:0])
	return h
}

func CheckForCoinSupplyOverflow(coin CalculateCoin, delta *big.Int) *Response {
	total := big.NewInt(0).Set(coin.Volume())
	total.Add(total, delta)

	if total.Cmp(coin.MaxSupply()) == 1 {
		return &Response{
			Code: code.CoinSupplyOverflow,
			Log:  "maximum supply reached",
			Info: EncodeError(code.NewCoinSupplyOverflow(delta.String(), coin.Volume().String(), total.String(), coin.MaxSupply().String(), coin.GetFullSymbol(), coin.ID().String())),
		}
	}

	return nil
}

func CheckReserveUnderflow(coin CalculateCoin, delta *big.Int) *Response {
	total := big.NewInt(0).Sub(coin.Reserve(), delta)

	if total.Cmp(minCoinReserve) == -1 {
		min := big.NewInt(0).Add(minCoinReserve, delta)
		return &Response{
			Code: code.CoinReserveUnderflow,
			Log:  fmt.Sprintf("coin %s reserve is too small (%s, required at least %s)", coin.GetFullSymbol(), coin.Reserve().String(), min.String()),
			Info: EncodeError(code.NewCoinReserveUnderflow(delta.String(), coin.Reserve().String(), total.String(), minCoinReserve.String(), coin.GetFullSymbol(), coin.ID().String())),
		}
	}

	return nil
}
