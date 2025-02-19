package http_adapter

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

func (h *HTTP) healthCheck(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "The server is responding ok")
}

func (h *HTTP) getIDByPatent(w http.ResponseWriter, r *http.Request) {
	patent := r.PathValue("patente")
	err, id := h.app.PatentToID(patent)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]int{
		"id": int(id),
	})
}

func (h *HTTP) getPatentByID(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
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

	err, patente := h.app.IDtoPatent(uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{
		"patente": patente,
	})
}
