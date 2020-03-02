package dao

import (
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/types"
)

var (
	Address = (&accounts.Multisig{
		Threshold: 2,
		Weights:   []uint{1, 1, 1},
		Addresses: []types.Address{types.HexToAddress("Mxed2f3dbe7a25f928df95ae8f207ed8079578daf3"), types.HexToAddress("Mx91980bf6391eb6946f43df559fd1e56952f9cde7"), types.HexToAddress("Mx375bc810e0fd19dcf0da43556b50ba6825ba11b8")},
	}).Address()
	Commission = 10
)
