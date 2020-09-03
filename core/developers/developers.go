package developers

import (
	"github.com/MinterTeam/minter-go-node/core/types"
)

// Commission which is subtracted from rewards and being send to Developers Address
var (
	Address    = types.HexToAddress("Mx688568d9d70c57e71d0b9de6480afb0d317f885c")
	Commission = 10 // in %
)
