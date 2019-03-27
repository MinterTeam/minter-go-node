package main

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/genesis"
)

func main() {
	gen, _ := genesis.GetTestnetGenesis()
	genesisJson, _ := json.MarshalIndent(gen, "", "	")
	println(string(genesisJson))
}
