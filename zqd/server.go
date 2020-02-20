package zqd

import (
	"encoding/json"
	"net/http"

	"github.com/brimsec/zq/zqd/pcap"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/gorilla/mux"
)

type VersionMessage struct {
	Zqd string `json:"boomd"` //XXX boomd -> zqd
	Zq  string `json:"zq"`
}

// This struct filled in by main from linker setting version strings.
var Version VersionMessage

func NewHandler() http.Handler {
	r := mux.NewRouter()
	space.AddRoutes(r)
	search.AddRoutes(r)
	pcap.AddRoutes(r)
	r.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	return r
}
