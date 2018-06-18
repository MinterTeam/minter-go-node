package api

import (
	"encoding/json"
	"github.com/MinterTeam/minter-go-node/core/rewards"
	"github.com/MinterTeam/minter-go-node/helpers"
	"math/big"
	"net/http"
	"strconv"
)

func GetBipVolume(w http.ResponseWriter, r *http.Request) {

	height, _ := strconv.Atoi(r.URL.Query().Get("height"))

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if height < 1 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Code: 400,
			Log:  "Provide height",
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: CalcBipVolume(height).String(),
	})
}

// TODO: change logic
// should we save amount to state?
func CalcBipVolume(height int) *big.Int {
	vol := big.NewInt(0)

	premine := helpers.BipToPip(big.NewInt(20000000)) // testnet's premine 20 000 000 bip

	vol.Add(vol, premine)

	for i := 1; i <= height; i++ {
		vol.Add(vol, rewards.GetRewardForBlock(uint64(height)))

		if height <= 3110400*6 && height%3110400 == 0 { // team distribution
			value := big.NewInt(300000000) // 300 000 000 bip (3%)
			vol.Add(vol, helpers.BipToPip(value))
		}

		if height <= 3110400*10 && height%3110400 == 0 { // airdrop distribution
			value := big.NewInt(500000000) // 500 000 000 bip (5%)
			vol.Add(vol, helpers.BipToPip(value))
		}
	}

	return vol
}
