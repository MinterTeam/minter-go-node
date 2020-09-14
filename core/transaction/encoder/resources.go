package encoder

import (
	"encoding/base64"
	"strconv"

	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
)

// TxDataResource is an interface for preparing JSON representation of TxData
type TxDataResource interface {
	Transform(txData interface{}, context *state.CheckState) TxDataResource
}

// CoinResource is a JSON representation of a coin
type CoinResource struct {
	ID     uint32 `json:"id"`
	Symbol string `json:"symbol"`
}

// SendDataResource is JSON representation of TxType 0x01
type SendDataResource struct {
	Coin  CoinResource `json:"coin"`
	To    string       `json:"to"`
	Value string       `json:"value"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (SendDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.SendData)
	coin := context.Coins().GetCoin(data.Coin)

	return SendDataResource{
		To:    data.To.String(),
		Value: data.Value.String(),
		Coin:  CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// SellCoinDataResource is JSON representation of TxType 0x02
type SellCoinDataResource struct {
	CoinToSell        CoinResource `json:"coin_to_sell"`
	ValueToSell       string       `json:"value_to_sell"`
	CoinToBuy         CoinResource `json:"coin_to_buy"`
	MinimumValueToBuy string       `json:"minimum_value_to_buy"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (SellCoinDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.SellCoinData)
	buyCoin := context.Coins().GetCoin(data.CoinToBuy)
	sellCoin := context.Coins().GetCoin(data.CoinToSell)

	return SellCoinDataResource{
		ValueToSell:       data.ValueToSell.String(),
		MinimumValueToBuy: data.MinimumValueToBuy.String(),
		CoinToBuy:         CoinResource{buyCoin.ID().Uint32(), buyCoin.GetFullSymbol()},
		CoinToSell:        CoinResource{sellCoin.ID().Uint32(), sellCoin.GetFullSymbol()},
	}
}

// SellAllCoinDataResource is JSON representation of TxType 0x03
type SellAllCoinDataResource struct {
	CoinToSell        CoinResource `json:"coin_to_sell"`
	CoinToBuy         CoinResource `json:"coin_to_buy"`
	MinimumValueToBuy string       `json:"minimum_value_to_buy"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (SellAllCoinDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.SellAllCoinData)
	buyCoin := context.Coins().GetCoin(data.CoinToBuy)
	sellCoin := context.Coins().GetCoin(data.CoinToSell)

	return SellAllCoinDataResource{
		MinimumValueToBuy: data.MinimumValueToBuy.String(),
		CoinToBuy:         CoinResource{buyCoin.ID().Uint32(), buyCoin.GetFullSymbol()},
		CoinToSell:        CoinResource{sellCoin.ID().Uint32(), sellCoin.GetFullSymbol()},
	}
}

// BuyCoinDataResource is JSON representation of TxType 0x04
type BuyCoinDataResource struct {
	CoinToBuy          CoinResource `json:"coin_to_buy"`
	ValueToBuy         string       `json:"value_to_buy"`
	CoinToSell         CoinResource `json:"coin_to_sell"`
	MaximumValueToSell string       `json:"maximum_value_to_sell"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (BuyCoinDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.BuyCoinData)
	buyCoin := context.Coins().GetCoin(data.CoinToBuy)
	sellCoin := context.Coins().GetCoin(data.CoinToSell)

	return BuyCoinDataResource{
		ValueToBuy:         data.ValueToBuy.String(),
		MaximumValueToSell: data.MaximumValueToSell.String(),
		CoinToBuy:          CoinResource{buyCoin.ID().Uint32(), buyCoin.GetFullSymbol()},
		CoinToSell:         CoinResource{sellCoin.ID().Uint32(), sellCoin.GetFullSymbol()},
	}
}

// CreateCoinDataResource is JSON representation of TxType 0x05
type CreateCoinDataResource struct {
	Name                 string `json:"name"`
	Symbol               string `json:"symbol"`
	InitialAmount        string `json:"initial_amount"`
	InitialReserve       string `json:"initial_reserve"`
	ConstantReserveRatio string `json:"constant_reserve_ratio"`
	MaxSupply            string `json:"max_supply"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (CreateCoinDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.CreateCoinData)

	return CreateCoinDataResource{
		Name:                 data.Name,
		Symbol:               data.Symbol.String(),
		InitialAmount:        data.InitialAmount.String(),
		InitialReserve:       data.InitialReserve.String(),
		ConstantReserveRatio: strconv.Itoa(int(data.ConstantReserveRatio)),
		MaxSupply:            data.MaxSupply.String(),
	}
}

// DeclareCandidacyDataResource is JSON representation of TxType 0x06
type DeclareCandidacyDataResource struct {
	Address    string       `json:"address"`
	PubKey     string       `json:"pub_key"`
	Commission string       `json:"commission"`
	Coin       CoinResource `json:"coin"`
	Stake      string       `json:"stake"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (DeclareCandidacyDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.DeclareCandidacyData)
	coin := context.Coins().GetCoin(data.Coin)

	return DeclareCandidacyDataResource{
		Address:    data.Address.String(),
		PubKey:     data.PubKey.String(),
		Commission: strconv.Itoa(int(data.Commission)),
		Stake:      data.Stake.String(),
		Coin:       CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// DelegateDataResource is JSON representation of TxType 0x07
type DelegateDataResource struct {
	PubKey string       `json:"pub_key"`
	Coin   CoinResource `json:"coin"`
	Value  string       `json:"value"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (DelegateDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.DelegateData)
	coin := context.Coins().GetCoin(data.Coin)

	return DelegateDataResource{
		PubKey: data.PubKey.String(),
		Value:  data.Value.String(),
		Coin:   CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// UnbondDataResource is JSON representation of TxType 0x08
type UnbondDataResource struct {
	PubKey string       `json:"pub_key"`
	Coin   CoinResource `json:"coin"`
	Value  string       `json:"value"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (UnbondDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.UnbondData)
	coin := context.Coins().GetCoin(data.Coin)

	return UnbondDataResource{
		PubKey: data.PubKey.String(),
		Value:  data.Value.String(),
		Coin:   CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// RedeemCheckDataResource is JSON representation of TxType 0x09
type RedeemCheckDataResource struct {
	RawCheck string `json:"raw_check"`
	Proof    string `json:"proof"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (RedeemCheckDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.RedeemCheckData)

	return RedeemCheckDataResource{
		RawCheck: base64.StdEncoding.EncodeToString(data.RawCheck),
		Proof:    base64.StdEncoding.EncodeToString(data.Proof[:]),
	}
}

// SetCandidateOnDataResource is JSON representation of TxType 0x0A
type SetCandidateOnDataResource struct {
	PubKey string `json:"pub_key"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (SetCandidateOnDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.SetCandidateOnData)
	return SetCandidateOnDataResource{data.PubKey.String()}
}

// SetCandidateOffDataResource is JSON representation of TxType 0x0B
type SetCandidateOffDataResource struct {
	PubKey string `json:"pub_key"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (SetCandidateOffDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.SetCandidateOffData)
	return SetCandidateOffDataResource{data.PubKey.String()}
}

// CreateMultisigDataResource is JSON representation of TxType 0x0C
type CreateMultisigDataResource struct {
	Threshold string          `json:"threshold"`
	Weights   []string        `json:"weights"`
	Addresses []types.Address `json:"addresses"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (CreateMultisigDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.CreateMultisigData)

	var weights []string
	for _, weight := range data.Weights {
		weights = append(weights, strconv.Itoa(int(weight)))
	}

	return CreateMultisigDataResource{
		Threshold: strconv.Itoa(int(data.Threshold)),
		Weights:   weights,
		Addresses: data.Addresses,
	}
}

// MultiSendDataResource is JSON representation of TxType 0x0D
type MultiSendDataResource struct {
	List []SendDataResource `json:"list"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (resource MultiSendDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.MultisendData)

	for _, v := range data.List {
		coin := context.Coins().GetCoin(v.Coin)

		resource.List = append(resource.List, SendDataResource{
			Coin:  CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
			To:    v.To.String(),
			Value: v.Value.String(),
		})
	}

	return resource
}

// EditCandidateDataResource is JSON representation of TxType 0x0E
type EditCandidateDataResource struct {
	PubKey         string  `json:"pub_key"`
	NewPubKey      *string `json:"new_pub_key"`
	RewardAddress  string  `json:"reward_address"`
	OwnerAddress   string  `json:"owner_address"`
	ControlAddress string  `json:"control_address"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (EditCandidateDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.EditCandidateData)
	return EditCandidateDataResource{
		PubKey:         data.PubKey.String(),
		RewardAddress:  data.RewardAddress.String(),
		OwnerAddress:   data.OwnerAddress.String(),
		ControlAddress: data.ControlAddress.String(),
	}
}

// SetHaltBlockDataResource is JSON representation of TxType 0x0F
type SetHaltBlockDataResource struct {
	PubKey string `json:"pub_key"`
	Height string `json:"height"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (SetHaltBlockDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.SetHaltBlockData)

	return SetHaltBlockDataResource{
		PubKey: data.PubKey.String(),
		Height: strconv.FormatUint(data.Height, 10),
	}
}

// RecreateCoinDataResource is JSON representation of TxType 0x10
type RecreateCoinDataResource struct {
	Name                 string           `json:"name"`
	Symbol               types.CoinSymbol `json:"symbol"`
	InitialAmount        string           `json:"initial_amount"`
	InitialReserve       string           `json:"initial_reserve"`
	ConstantReserveRatio string           `json:"constant_reserve_ratio"`
	MaxSupply            string           `json:"max_supply"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (RecreateCoinDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.RecreateCoinData)

	return RecreateCoinDataResource{
		Name:                 data.Name,
		Symbol:               data.Symbol,
		InitialAmount:        data.InitialAmount.String(),
		InitialReserve:       data.InitialReserve.String(),
		ConstantReserveRatio: strconv.Itoa(int(data.ConstantReserveRatio)),
		MaxSupply:            data.MaxSupply.String(),
	}
}

// EditCoinOwnerDataResource is JSON representation of TxType 0x11
type EditCoinOwnerDataResource struct {
	Symbol   types.CoinSymbol `json:"symbol"`
	NewOwner types.Address    `json:"new_owner"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (EditCoinOwnerDataResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.EditCoinOwnerData)

	return EditCoinOwnerDataResource{
		Symbol:   data.Symbol,
		NewOwner: data.NewOwner,
	}
}

// EditMultisigOwnersResource is JSON representation of TxType 0x12
type EditMultisigOwnersResource struct {
	Threshold string          `json:"threshold"`
	Weights   []string        `json:"weights"`
	Addresses []types.Address `json:"addresses"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (EditMultisigOwnersResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.EditMultisigOwnersData)

	resource := EditMultisigOwnersResource{
		Addresses: data.Addresses,
		Threshold: strconv.Itoa(int(data.Threshold)),
	}

	resource.Weights = make([]string, 0, len(data.Weights))
	for _, weight := range data.Weights {
		resource.Weights = append(resource.Weights, strconv.Itoa(int(weight)))
	}

	return resource
}

// PriceVoteResource is JSON representation of TxType 0x13
type PriceVoteResource struct {
	Price uint32 `json:"price"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (PriceVoteResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.PriceVoteData)

	return PriceVoteResource{
		Price: uint32(data.Price),
	}
}

// EditCandidatePublicKeyResource is JSON representation of TxType 0x14
type EditCandidatePublicKeyResource struct {
	PubKey    string `json:"pub_key"`
	NewPubKey string `json:"new_pub_key"`
}

// Transform returns TxDataResource from given txData. Used for JSON encoder.
func (EditCandidatePublicKeyResource) Transform(txData interface{}, context *state.CheckState) TxDataResource {
	data := txData.(*transaction.EditCandidatePublicKeyData)

	return EditCandidatePublicKeyResource{
		PubKey:    data.PubKey.String(),
		NewPubKey: data.NewPubKey.String(),
	}
}
