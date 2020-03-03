package developers

import (
	"github.com/MinterTeam/minter-go-node/core/state/accounts"
	"github.com/MinterTeam/minter-go-node/core/types"
)

var (
	Address = (&accounts.Multisig{
		Threshold: 2,
		Weights:   []uint{1, 1, 1},
		Addresses: []types.Address{types.HexToAddress("Mx22df0f98c1b421974fb0c64440258aaffd4e96d8"), types.HexToAddress("Mxf8821646818a873e3efac40cc2c13f96e3515aa1"), types.HexToAddress("Mx90a82ed6fd69cdd125474ce9349a8e34fb4f5ffe")},
	}).Address()
	Commission = 10
)
