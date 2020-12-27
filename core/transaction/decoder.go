package transaction

import (
	"errors"
	"fmt"
	"github.com/MinterTeam/minter-go-node/rlp"
)

func getData(txType TxType) (Data, bool) {
	switch txType {
	case TypeSend:
		return &SendData{}, true
	case TypeSellCoin:
		return &SellCoinData{}, true
	case TypeSellAllCoin:
		return &SellAllCoinData{}, true
	case TypeBuyCoin:
		return &BuyCoinData{}, true
	case TypeCreateCoin:
		return &CreateCoinData{}, true
	case TypeDeclareCandidacy:
		return &DeclareCandidacyData{}, true
	case TypeDelegate:
		return &DelegateData{}, true
	case TypeUnbond:
		return &UnbondData{}, true
	case TypeRedeemCheck:
		return &RedeemCheckData{}, true
	case TypeSetCandidateOnline:
		return &SetCandidateOnData{}, true
	case TypeSetCandidateOffline:
		return &SetCandidateOffData{}, true
	case TypeMultisend:
		return &MultisendData{}, true
	case TypeCreateMultisig:
		return &CreateMultisigData{}, true
	case TypeEditCandidate:
		return &EditCandidateData{}, true
	case TypeSetHaltBlock:
		return &SetHaltBlockData{}, true
	case TypeRecreateCoin:
		return &RecreateCoinData{}, true
	case TypeEditCoinOwner:
		return &EditCoinOwnerData{}, true
	case TypeEditMultisig:
		return &EditMultisigData{}, true
	case TypePriceVote:
		return &PriceVoteData{}, true
	case TypeEditCandidatePublicKey:
		return &EditCandidatePublicKeyData{}, true
	case TypeAddSwapPool:
		return &AddSwapPoolData{}, true
	case TypeRemoveSwapPool:
		return &RemoveSwapPoolData{}, true
	case TypeSellSwapPool:
		return &SellSwapPoolData{}, true
	case TypeBuySwapPool:
		return &BuySwapPoolData{}, true
	case TypeSellAllSwapPool:
		return &SellAllSwapPoolData{}, true
	case TypeEditCommission:
		return &EditCommissionData{}, true
	case TypeMoveStake:
		return &MoveStakeData{}, true
	case TypeMintCoin:
		return &MintCoin{}, true
	case TypeBurnCoin:
		return &BurnCoin{}, true
	default:
		return nil, false
	}
}

type Decoder struct {
	registeredTypes map[TxType]Data
}

func (decoder *Decoder) RegisterType(t TxType, d Data) {
	decoder.registeredTypes[t] = d
}

func DecodeFromBytes(buf []byte) (*Transaction, error) {
	tx, err := DecodeFromBytesWithoutSig(buf)
	if err != nil {
		return nil, err
	}

	tx, err = DecodeSig(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func DecodeSig(tx *Transaction) (*Transaction, error) {
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

	return tx, nil
}

func DecodeFromBytesWithoutSig(buf []byte) (*Transaction, error) {
	var tx Transaction
	err := rlp.DecodeBytes(buf, &tx)

	if err != nil {
		return nil, err
	}

	if tx.Data == nil {
		return nil, errors.New("incorrect tx data")
	}

	d, ok := getData(tx.Type)

	if !ok {
		return nil, fmt.Errorf("tx type %x is not registered", tx.Type)
	}

	err = rlp.DecodeBytes(tx.Data, d)

	if err != nil {
		return nil, err
	}

	tx.SetDecodedData(d)

	return &tx, nil
}
