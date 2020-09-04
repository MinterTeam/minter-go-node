package dao

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

// Commission which is subtracted from rewards and being send to DAO Address
var (
	Address    = types.HexToAddress("Mx7f0fc21d932f38ca9444f61703174569066cfa50")
	Commission = 10 // in %
)
