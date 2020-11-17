package exchange

import (
	"github.com/MinterTeam/minter-go-node/core/types"
	"math/big"
	"testing"
)

func TestUniswap_Balance(t *testing.T) {
	uniswap := NewUniswap()
	err := uniswap.Add(types.Address{1}, 0, big.NewInt(1e18), 1, big.NewInt(1e10))
	if err != nil {
		t.Fatal(err)
	}
	balance1, percent1, err := uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(balance1[0].String())
	t.Log(balance1[1].String())
	t.Log(percent1.String())
}

func TestUniswap_Add(t *testing.T) {
	uniswap := NewUniswap()
	err := uniswap.Add(types.Address{1}, 0, big.NewInt(100), 1, big.NewInt(10))
	if err != nil {
		t.Fatal(err)
	}

	balance1, percent1, err := uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	if balance1[0].String() != "100" {
		t.Log(balance1[0].String())
		t.Fatal("balance1")
	}
	if balance1[1].String() != "10" {
		t.Log(balance1[0].String())
		t.Fatal("balance1")
	}
	if percent1.String() != "100" {
		t.Log(percent1.String())
		t.Fatal("percent1")
	}

	err = uniswap.Add(types.Address{2}, 0, big.NewInt(100), 1, big.NewInt(10))
	if err != nil {
		t.Fatal(err)
	}

	balance2, percent2, err := uniswap.Balance(types.Address{2}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if balance2[0].String() != "100" {
		t.Log(balance2[0].String())
		t.Error("balance2")
	}
	if balance2[1].String() != "10" {
		t.Log(balance2[0].String())
		t.Error("balance2")
	}
	if percent2.String() != "50" {
		t.Log(percent2.String())
		t.Fatal("percent2")
	}

	balance1, percent1, err = uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if balance1[0].String() != "100" {
		t.Log(balance1[0].String())
		t.Error("balance1")
	}
	if percent1.String() != "50" {
		t.Log(percent1.String())
		t.Fatal("percent1")
	}

	err = uniswap.Add(types.Address{2}, 0, big.NewInt(100), 1, big.NewInt(10))
	if err != nil {
		t.Fatal(err)
	}

	balance1, percent1, err = uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if balance1[0].String() != "100" {
		t.Log(balance1[0].String())
		t.Error("balance1")
	}
	if balance1[1].String() != "10" {
		t.Log(balance1[1].String())
		t.Error("balance1")
	}
	if percent1.String() != "33.33333333" {
		t.Log(percent1.String())
		t.Fatal("percent1")
	}

	balance2, percent2, err = uniswap.Balance(types.Address{2}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if balance2[0].String() != "200" {
		t.Log(balance2[0].String())
		t.Fatal("balance2")
	}
	if balance2[1].String() != "20" {
		t.Log(balance2[0].String())
		t.Fatal("balance2")
	}
	if percent2.String() != "66.66666667" {
		t.Log(percent2.String())
		t.Fatal("percent2")
	}

}

func TestUniswap_Add_any(t *testing.T) {
	uniswap := NewUniswap()
	err := uniswap.Add(types.Address{1}, 0, big.NewInt(1e18), 1, big.NewInt(1e10))
	if err != nil {
		t.Fatal(err)
	}

	err = uniswap.Add(types.Address{2}, 1, big.NewInt(1e10), 2, big.NewInt(1e18))
	if err != nil {
		t.Fatal(err)
	}

	balance1, percent1, err := uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(balance1[0].String())
	t.Log(percent1.String())

	balance2, percent2, err := uniswap.Balance(types.Address{2}, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(balance2[0].String())
	t.Log(percent2.String())
}

func TestUniswap_Add_one(t *testing.T) {
	uniswap := NewUniswap()
	err := uniswap.Add(types.Address{1}, 0, big.NewInt(123), 1, big.NewInt(34))
	if err != nil {
		t.Fatal(err)
	}

	balance1, percent1, err := uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(balance1[0].String())
	t.Log(percent1.String())

	err = uniswap.Add(types.Address{1}, 0, big.NewInt(123), 1, big.NewInt(34))
	if err != nil {
		t.Fatal(err)
	}

	balance1, percent1, err = uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(balance1[0].String())
	t.Log(percent1.String())

	err = uniswap.Add(types.Address{1}, 0, big.NewInt(123), 1, big.NewInt(34))
	if err != nil {
		t.Fatal(err)
	}

	balance1, percent1, err = uniswap.Balance(types.Address{1}, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(balance1[0].String())
	t.Log(percent1.String())

}

func Test_percentToVolume(t *testing.T) {
	t.Log(percentToVolume(big.NewInt(1000), big.NewFloat(20)).String())
}
func Test_volumeToPercent(t *testing.T) {
	t.Log(volumeToPercent(big.NewInt(1000), big.NewInt(20)).String())
}
