package zqd

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/brimsec/zq/zqd/api"
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
	h.Handle("/space/{space}", handleSpacePut).Methods("PUT")
	h.Handle("/space/{space}", handleSpaceDelete).Methods("DELETE")
	h.Handle("/space/{space}/pcap", handlePcapSearch).Methods("GET")
	h.Handle("/space/{space}/pcap", handlePcapPost).Methods("POST")
	h.Handle("/space/{space}/log", handleLogPost).Methods("POST")
	h.Handle("/space/{space}/index", handleIndexPost).Methods("POST")
	h.Handle("/space/{space}/indexsearch", handleIndexSearch).Methods("POST")
	h.Handle("/space/{space}/archivestat", handleArchiveStat).Methods("GET")
	h.Handle("/space/{space}/subspace", handleSubspacePost).Methods("POST")
	h.Handle("/search", handleSearch).Methods("POST")
	h.Handle("/worker", handleWorker).Methods("POST")
	h.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&api.VersionResponse{Version: core.Version})
	})
	h.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	h.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, indexPage)
	})
	return h
}

const indexPage = `
<!DOCTYPE html>
<html>
  <title>ZQD daemon</title>
  <body style="padding:10px">
    <h2>ZQD</h2>
    <p>A <a href="https://github.com/brimsec/zq/tree/master/cmd/zqd">zqd</a> daemon is listening on this host/port.</p>
    <p>If you're a <a href="https://www.brimsecurity.com/">Brim</a> user, connect to this host/port from the <a href="https://github.com/brimsec/brim">Brim application</a> in the graphical desktop interface in your operating system (not a web browser).</p>
    <p>If your goal is to perform command line operations against this zqd, use the <a href="https://github.com/brimsec/zq/tree/master/cmd/zapi">zapi</a> client.</p>
  </body>
</html>`
