package http_adapter

func (h *HTTP) SetRoutes() {
	h.mux.HandleFunc("GET /patente/{id}", h.getPatentByID)
	h.mux.HandleFunc("GET /id/{patente}", h.getIDByPatent)
	h.mux.HandleFunc("GET /healthcheck", h.healthCheck)
}
