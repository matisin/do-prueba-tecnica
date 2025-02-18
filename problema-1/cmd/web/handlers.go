package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

func (web *Web) healthCheck(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "The server is responding ok")
}

func (web *Web) getPatenteByID(w http.ResponseWriter, r *http.Request) {
	idString := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idString)
	if err != nil {
		http.Error(w, "id must be a valid number", http.StatusBadRequest)
		return
	}
	if id < 0 {
		http.Error(w, "id must be greater than 0", http.StatusBadRequest)
		return
	}

	uid := uint(id)

	err, patente := web.app.GetPatente(uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"patente": patente,
	})
}
