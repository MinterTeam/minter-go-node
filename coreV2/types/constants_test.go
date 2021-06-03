package types

import (
	"testing"
)

func TestGetBaseCoin(t *testing.T) {
	CurrentChainID = ChainTestnet
	if GetBaseCoin().Compare(coinTestnet) != 0 {
		t.Error("Incorrect base coin")
	}

	CurrentChainID = ChainMainnet
	if GetBaseCoin().Compare(coinMainnet) != 0 {
		t.Error("Incorrect base coin")
	}
}

func TestGetBaseCoinID(t *testing.T) {
	if GetBaseCoinID() != BasecoinID {
		t.Error("Incorrect base coin id")
	}
}
