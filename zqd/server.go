package zqd

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type VersionMessage struct {
	Zqd string `json:"boomd"` //XXX boomd -> zqd
	Zq  string `json:"zq"`
}

// This struct filled in by main from linker setting version strings.
var Version VersionMessage

func NewHandler(root string) http.Handler {
	r := mux.NewRouter()
	r = r.UseEncodedPath()
	r.Handle("/space", wrapRoot(root, handleSpaceList)).Methods("GET")
	r.Handle("/space", wrapRoot(root, handleSpacePost)).Methods("POST")
	r.Handle("/space/{space}", wrapRoot(root, handleSpaceGet)).Methods("GET")
	r.Handle("/space/{space}", wrapRoot(root, handleSpaceDelete)).Methods("DELETE")
	r.Handle("/space/{space}/packet", wrapRoot(root, handlePacketSearch)).Methods("GET")
	r.Handle("/space/{space}/packet", wrapRoot(root, handlePacketPost)).Methods("POST")
	r.Handle("/search", wrapRoot(root, handleSearch)).Methods("POST")
	r.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	return r
}

type handlerFunc func(root string, w http.ResponseWriter, r *http.Request)

func wrapRoot(root string, h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h(root, w, r)
	})
}
