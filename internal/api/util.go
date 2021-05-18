package api

import (
	"encoding/json"
	"net/http"
)

func setJSONContentHeader(w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json")
}

func sendSerialized(w http.ResponseWriter, v interface{}) {
	raw, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setJSONContentHeader(w)
	w.Write(raw)
}