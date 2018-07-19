package api

import (
	"encoding/json"
	"net/http"
)

func NetInfo(w http.ResponseWriter, r *http.Request) {

	result, err := client.NetInfo()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if err != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Code:   500,
			Result: nil,
			Log:    err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(Response{
		Code:   0,
		Result: result,
	})
}
