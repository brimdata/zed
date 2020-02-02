package zqd

import (
	"net/http"

	"github.com/mccanne/zq/zqd/search"
	"github.com/mccanne/zq/zqd/space"
)

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
	return http.ListenAndServe(port, nil)
}
