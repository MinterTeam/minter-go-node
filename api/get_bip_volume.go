package api

import (
	"encoding/json"
	"net/http"
)

type BipVolumeResult struct {
	Volume string `json:"volume"`
}

func GetBipVolume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(Response{
		Code: 501,
		Log:  "Not implemented",
	})
	return
}