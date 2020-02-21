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

func NewHandler() http.Handler {
	r := mux.NewRouter()
	r = r.UseEncodedPath()
	r.HandleFunc("/space/{space}/", SpaceInfoEndpoint).Methods("GET")
	r.HandleFunc("/space/{space}/packet/", PacketSearchEndpoint).Methods("GET")
	r.HandleFunc("/search/", SearchEndpoint).Methods("POST")
	r.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	return r
}
