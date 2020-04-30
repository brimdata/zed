package zqd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type handlerFunc func(root *Core, w http.ResponseWriter, r *http.Request)

type handler struct {
	*mux.Router
	core *Core
}

func (h *handler) Handle(path string, f handlerFunc) *mux.Route {
	return h.Router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		f(h.core, w, r)
	})
}

func NewHandler(core *Core, logger *zap.Logger) http.Handler {
	h := handler{Router: mux.NewRouter(), core: core}
	h.Use(requestIDMiddleware())
	h.Use(accessLogMiddleware(logger))
	h.Use(panicCatchMiddleware(logger))
	h.Handle("/space", handleSpaceList).Methods("GET")
	h.Handle("/space", handleSpacePost).Methods("POST")
	h.Handle("/space/{space}", handleSpaceGet).Methods("GET")
	h.Handle("/space/{space}", handleSpaceDelete).Methods("DELETE")
	h.Handle("/space/{space}/pcap", handlePcapSearch).Methods("GET")
	h.Handle("/space/{space}/pcap", handlePcapPost).Methods("POST")
	h.Handle("/space/{space}/log", handleLogPost).Methods("POST")
	h.Handle("/search", handleSearch).Methods("POST")
	h.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	h.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	// XXX This can be removed once a new release has been cut and added to
	// brim.
	h.HandleFunc("/space/{space}/packet", func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["space"]
		http.Redirect(w, r, fmt.Sprintf("/space/%s/pcap", name), http.StatusPermanentRedirect)
	})
	return h
}
