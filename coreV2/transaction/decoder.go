package transaction

import (
	"errors"
	"fmt"

	"github.com/MinterTeam/minter-go-node/rlp"
)

func GetDataV1(txType TxType) (Data, bool) {
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
	case TypeEditCandidatePublicKey:
		return &EditCandidatePublicKeyData{}, true
	case TypeAddLiquidity:
		return &AddLiquidityData{}, true
	case TypeRemoveLiquidity:
		return &RemoveLiquidityV1{}, true
	case TypeSellSwapPool:
		return &SellSwapPoolDataV1{}, true
	case TypeBuySwapPool:
		return &BuySwapPoolDataV1{}, true
	case TypeSellAllSwapPool:
		return &SellAllSwapPoolDataV1{}, true
	case TypeEditCandidateCommission:
		return &EditCandidateCommission{}, true
	case TypeMintToken:
		return &MintTokenData{}, true
	case TypeBurnToken:
		return &BurnTokenData{}, true
	case TypeCreateToken:
		return &CreateTokenData{}, true
	case TypeRecreateToken:
		return &RecreateTokenData{}, true
	case TypeVoteCommission:
		return &VoteCommissionData{}, true
	case TypeVoteUpdate:
		return &VoteUpdateDataV1{}, true
	case TypeCreateSwapPool:
		return &CreateSwapPoolData{}, true
	default:
		return nil, false
	}
}

func GetData(txType TxType) (Data, bool) { return GetDataV240(txType) }

func GetDataV240(txType TxType) (Data, bool) {
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
	case TypeEditCandidatePublicKey:
		return &EditCandidatePublicKeyData{}, true
	case TypeAddLiquidity:
		return &AddLiquidityData{}, true
	case TypeRemoveLiquidity:
		return &RemoveLiquidityV230{}, true
	case TypeSellSwapPool:
		return &SellSwapPoolDataV240{}, true
	case TypeBuySwapPool:
		return &BuySwapPoolDataV240{}, true
	case TypeSellAllSwapPool:
		return &SellAllSwapPoolDataV240{}, true
	case TypeEditCandidateCommission:
		return &EditCandidateCommission{}, true
	case TypeMintToken:
		return &MintTokenData{}, true
	case TypeBurnToken:
		return &BurnTokenData{}, true
	case TypeCreateToken:
		return &CreateTokenData{}, true
	case TypeRecreateToken:
		return &RecreateTokenData{}, true
	case TypeVoteCommission:
		return &VoteCommissionDataV240{}, true
	case TypeVoteUpdate:
		return &VoteUpdateDataV230{}, true
	case TypeCreateSwapPool:
		return &CreateSwapPoolData{}, true
	default:
		return nil, false
	}
}

func GetDataV230(txType TxType) (Data, bool) {
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
	case TypeEditCandidatePublicKey:
		return &EditCandidatePublicKeyData{}, true
	case TypeAddLiquidity:
		return &AddLiquidityData{}, true
	case TypeRemoveLiquidity:
		return &RemoveLiquidityV230{}, true
	case TypeSellSwapPool:
		return &SellSwapPoolDataV230{}, true
	case TypeBuySwapPool:
		return &BuySwapPoolDataV230{}, true
	case TypeSellAllSwapPool:
		return &SellAllSwapPoolDataV230{}, true
	case TypeEditCandidateCommission:
		return &EditCandidateCommission{}, true
	case TypeMintToken:
		return &MintTokenData{}, true
	case TypeBurnToken:
		return &BurnTokenData{}, true
	case TypeCreateToken:
		return &CreateTokenData{}, true
	case TypeRecreateToken:
		return &RecreateTokenData{}, true
	case TypeVoteCommission:
		return &VoteCommissionData{}, true
	case TypeVoteUpdate:
		return &VoteUpdateDataV230{}, true
	case TypeCreateSwapPool:
		return &CreateSwapPoolData{}, true
	default:
		return nil, false
	}
}

func (e *Executor) DecodeFromBytes(buf []byte) (*Transaction, error) {
	tx, err := e.DecodeFromBytesWithoutSig(buf)
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

func (e *Executor) DecodeFromBytesWithoutSig(buf []byte) (*Transaction, error) {
	var tx Transaction
	err := rlp.DecodeBytes(buf, &tx)

	if err != nil {
		return nil, err
	}

	if tx.Data == nil {
		return nil, errors.New("incorrect tx data")
	}

	d, ok := e.decodeTxFunc(tx.Type)

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
