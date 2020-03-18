package zqd

import (
	"encoding/json"
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

func NewHandler(root *Core) http.Handler {
	return NewHandlerWithLogger(root, root.logger)
}

func NewHandlerWithLogger(core *Core, logger *zap.Logger) http.Handler {
	h := handler{Router: mux.NewRouter(), core: core}
	h.Use(requestIDMiddleware())
	h.Use(accessLogMiddleware(logger))
	h.Handle("/space", handleSpaceList).Methods("GET")
	h.Handle("/space", handleSpacePost).Methods("POST")
	h.Handle("/space/{space}", handleSpaceGet).Methods("GET")
	h.Handle("/space/{space}", handleSpaceDelete).Methods("DELETE")
	h.Handle("/space/{space}/packet", handlePacketSearch).Methods("GET")
	// Packet post endpoint is dependent on having zeek. If zeek is not
	// supported disable endpoint.
	if core.HasZeek() {
		h.Handle("/space/{space}/packet", handlePacketPost).Methods("POST")
	}
	h.Handle("/search", handleSearch).Methods("POST")
	h.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	h.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	return h
}
