package encoder

import (
	"encoding/base64"
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/types"
	"strconv"
)

type TxDataResource interface {
	Transform(txData interface{}, state *state.CheckState) TxDataResource
}

type CoinResource struct {
	ID     uint32 `json:"id"`
	Symbol string `json:"symbol"`
}

// TxType 0x01

type SendDataResource struct {
	Coin  CoinResource `json:"coin"`
	To    string       `json:"to"`
	Value string       `json:"value"`
}

func (SendDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.SendData)
	coin := state.Coins().GetCoin(data.Coin)

	return SendDataResource{
		To:    data.To.String(),
		Value: data.Value.String(),
		Coin:  CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// TxType 0x02

type SellCoinDataResource struct {
	CoinToSell        CoinResource `json:"coin_to_sell"`
	ValueToSell       string       `json:"value_to_sell"`
	CoinToBuy         CoinResource `json:"coin_to_buy"`
	MinimumValueToBuy string       `json:"minimum_value_to_buy"`
}

func (SellCoinDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.SellCoinData)
	buyCoin := state.Coins().GetCoin(data.CoinToBuy)
	sellCoin := state.Coins().GetCoin(data.CoinToSell)

	return SellCoinDataResource{
		ValueToSell:       data.ValueToSell.String(),
		MinimumValueToBuy: data.MinimumValueToBuy.String(),
		CoinToBuy:         CoinResource{buyCoin.ID().Uint32(), buyCoin.GetFullSymbol()},
		CoinToSell:        CoinResource{sellCoin.ID().Uint32(), sellCoin.GetFullSymbol()},
	}
}

// TxType 0x03

type SellAllCoinDataResource struct {
	CoinToSell        CoinResource `json:"coin_to_sell"`
	CoinToBuy         CoinResource `json:"coin_to_buy"`
	MinimumValueToBuy string       `json:"minimum_value_to_buy"`
}

func (SellAllCoinDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.SellAllCoinData)
	buyCoin := state.Coins().GetCoin(data.CoinToBuy)
	sellCoin := state.Coins().GetCoin(data.CoinToSell)

	return SellAllCoinDataResource{
		MinimumValueToBuy: data.MinimumValueToBuy.String(),
		CoinToBuy:         CoinResource{buyCoin.ID().Uint32(), buyCoin.GetFullSymbol()},
		CoinToSell:        CoinResource{sellCoin.ID().Uint32(), sellCoin.GetFullSymbol()},
	}
}

// TxType 0x04

type BuyCoinDataResource struct {
	CoinToBuy          CoinResource `json:"coin_to_buy"`
	ValueToBuy         string       `json:"value_to_buy"`
	CoinToSell         CoinResource `json:"coin_to_sell"`
	MaximumValueToSell string       `json:"maximum_value_to_sell"`
}

func (BuyCoinDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.BuyCoinData)
	buyCoin := state.Coins().GetCoin(data.CoinToBuy)
	sellCoin := state.Coins().GetCoin(data.CoinToSell)

	return BuyCoinDataResource{
		ValueToBuy:         data.ValueToBuy.String(),
		MaximumValueToSell: data.MaximumValueToSell.String(),
		CoinToBuy:          CoinResource{buyCoin.ID().Uint32(), buyCoin.GetFullSymbol()},
		CoinToSell:         CoinResource{sellCoin.ID().Uint32(), sellCoin.GetFullSymbol()},
	}
}

// TxType 0x05

type CreateCoinDataResource struct {
	Name                 string `json:"name"`
	Symbol               string `json:"symbol"`
	InitialAmount        string `json:"initial_amount"`
	InitialReserve       string `json:"initial_reserve"`
	ConstantReserveRatio string `json:"constant_reserve_ratio"`
	MaxSupply            string `json:"max_supply"`
}

func (CreateCoinDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
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

// TxType 0x06

type DeclareCandidacyDataResource struct {
	Address    string       `json:"address"`
	PubKey     string       `json:"pub_key"`
	Commission string       `json:"commission"`
	Coin       CoinResource `json:"coin"`
	Stake      string       `json:"stake"`
}

func (DeclareCandidacyDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.DeclareCandidacyData)
	coin := state.Coins().GetCoin(data.Coin)

	return DeclareCandidacyDataResource{
		Address:    data.Address.String(),
		PubKey:     data.PubKey.String(),
		Commission: strconv.Itoa(int(data.Commission)),
		Stake:      data.Stake.String(),
		Coin:       CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// TxType 0x07

type DelegateDataResource struct {
	PubKey string       `json:"pub_key"`
	Coin   CoinResource `json:"coin"`
	Value  string       `json:"value"`
}

func (DelegateDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.DelegateData)
	coin := state.Coins().GetCoin(data.Coin)

	return DelegateDataResource{
		PubKey: data.PubKey.String(),
		Value:  data.Value.String(),
		Coin:   CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// TxType 0x08

type UnbondDataResource struct {
	PubKey string       `json:"pub_key"`
	Coin   CoinResource `json:"coin"`
	Value  string       `json:"value"`
}

func (UnbondDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.UnbondData)
	coin := state.Coins().GetCoin(data.Coin)

	return UnbondDataResource{
		PubKey: data.PubKey.String(),
		Value:  data.Value.String(),
		Coin:   CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
	}
}

// TxType 0x09

type RedeemCheckDataResource struct {
	RawCheck string `json:"raw_check"`
	Proof    string `json:"proof"`
}

func (RedeemCheckDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.RedeemCheckData)

	return RedeemCheckDataResource{
		RawCheck: base64.StdEncoding.EncodeToString(data.RawCheck),
		Proof:    base64.StdEncoding.EncodeToString(data.Proof[:]),
	}
}

// TxType 0x0A

type SetCandidateOnDataResource struct {
	PubKey string `json:"pub_key"`
}

func (SetCandidateOnDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.SetCandidateOnData)
	return SetCandidateOnDataResource{data.PubKey.String()}
}

// TxType 0x0B

type SetCandidateOffDataResource struct {
	PubKey string `json:"pub_key"`
}

func (SetCandidateOffDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.SetCandidateOffData)
	return SetCandidateOffDataResource{data.PubKey.String()}
}

// TxType 0x0C

type CreateMultisigDataResource struct {
	Threshold string          `json:"threshold"`
	Weights   []string        `json:"weights"`
	Addresses []types.Address `json:"addresses"`
}

func (CreateMultisigDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
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

// TxType 0x0D

type MultiSendDataResource struct {
	List []SendDataResource `json:"list"`
}

func (resource MultiSendDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.MultisendData)

	for _, v := range data.List {
		coin := state.Coins().GetCoin(v.Coin)

		resource.List = append(resource.List, SendDataResource{
			Coin:  CoinResource{coin.ID().Uint32(), coin.GetFullSymbol()},
			To:    v.To.String(),
			Value: v.Value.String(),
		})
	}

	return resource
}

// TxType 0x0E

type EditCandidateDataResource struct {
	PubKey        string `json:"pub_key"`
	RewardAddress string `json:"reward_address"`
	OwnerAddress  string `json:"owner_address"`
}

func (EditCandidateDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.EditCandidateData)

	return EditCandidateDataResource{
		PubKey:        data.PubKey.String(),
		RewardAddress: data.RewardAddress.String(),
		OwnerAddress:  data.OwnerAddress.String(),
	}
}

// TxType 0x0F

type SetHaltBlockDataResource struct {
	PubKey string `json:"pub_key"`
	Height string `json:"height"`
}

func (SetHaltBlockDataResource) Transform(txData interface{}, state *state.CheckState) TxDataResource {
	data := txData.(*transaction.SetHaltBlockData)

	return SetHaltBlockDataResource{
		PubKey: data.PubKey.String(),
		Height: strconv.FormatUint(data.Height, 10),
	}
}
