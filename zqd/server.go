package zqd

import (
	"encoding/json"
	"net/http"

	"github.com/mccanne/zq/zqd/search"
	"github.com/mccanne/zq/zqd/space"
)

type VersionMessage struct {
	Zqd string `json:"boomd"` //XXX boomd -> zqd
	Zq  string `json:"zq"`
}

// This struct filled in by main from linker setting version strings.
var Version VersionMessage

func Run(port string) error {
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		search.Handle(w, r)
	})
	http.HandleFunc("/space", func(w http.ResponseWriter, r *http.Request) {
		space.HandleList(w, r)
	})
	http.HandleFunc("/space/", func(w http.ResponseWriter, r *http.Request) {
		space.HandleInfo(w, r)
	})
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&Version)
	})
	return http.ListenAndServe(port, nil)
}
