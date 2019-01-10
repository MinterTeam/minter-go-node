package api

import "math/big"

func MinGasPrice() (*big.Int, error) {
	return blockchain.MinGasPrice(), nil
}
