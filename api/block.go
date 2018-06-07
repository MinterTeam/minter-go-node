package api

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"minter/tmtypes"
	"net/http"
	"strconv"
)

// Single block (with meta)
type ResultBlock struct {
	BlockMeta *tmtypes.BlockMeta `json:"block_meta"`
	Block     *tmtypes.Block     `json:"block"`
}

func Block(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	height, _ := strconv.ParseInt(vars["height"], 10, 64)

	result := new(ResultBlock)
	_, err := client.Call("block", map[string]interface{}{
		"height": height,
	}, result)

	if err != nil {
		panic(err)
	}

	// TODO: check error

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result,
	})

	if err != nil {
		panic(err)
	}
}
