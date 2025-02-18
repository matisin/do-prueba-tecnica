package main

func (w *Web) SetRoutes() {
	w.mux.HandleFunc("GET /patente/{id}", w.getPatenteByID)
	w.mux.HandleFunc("GET /id/{patente}", w.getIDByPatente)
}
