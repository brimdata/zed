package space

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/pcap"
	"github.com/gorilla/mux"
)

func AddRoutes(router *mux.Router) {
	router = router.UseEncodedPath() // Allow url encoded spaces
	router.HandleFunc("/space/{space}/", HandleInfo).Methods("GET")
}

func spaceInfo(path string) (*api.SpaceInfo, error) {
	bzngFile := filepath.Join(path, "all.bzng")
	info, err := os.Stat(bzngFile)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(bzngFile)
	if err != nil {
		return nil, err
	}
	reader, err := detector.LookupReader("bzng", f, resolver.NewContext())
	if err != nil {
		return nil, err
	}
	minTs := nano.MaxTs
	maxTs := nano.MinTs
	var found bool
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		ts := rec.Ts
		if ts < minTs {
			minTs = ts
		}
		if ts > maxTs {
			maxTs = ts
		}
		found = true
	}
	s := &api.SpaceInfo{
		Name:          path,
		Size:          info.Size(),
		PacketSupport: pcap.HasPcaps(path),
	}
	if found {
		s.MinTime = &minTs
		s.MaxTime = &maxTs
	}
	return s, nil
}

func HandleInfo(w http.ResponseWriter, r *http.Request) {
	space, err := api.ExtractSpace(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// XXX this is slow.  can easily cache result rather than scanning
	// whole file each time.
	info, err := spaceInfo(space)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
