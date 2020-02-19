package zqd

import (
	"encoding/json"
	"net/http"

	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
)

type VersionMessage struct {
	Zqd string `json:"boomd"` //XXX boomd -> zqd
	Zq  string `json:"zq"`
}

// This struct filled in by main from linker setting version strings.
var Version VersionMessage

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		search.Handle(w, r)
	})
	mux.HandleFunc("/space/", func(w http.ResponseWriter, r *http.Request) {
		space.HandleInfo(w, r)
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	return mux
}
