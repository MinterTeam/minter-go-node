package bus

import "math/big"

type App interface {
	AddTotalSlashed(*big.Int)
}
