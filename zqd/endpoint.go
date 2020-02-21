package zqd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/pcap"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/gorilla/mux"
)

func SearchEndpoint(w http.ResponseWriter, r *http.Request) {
	var req api.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var out search.Output
	format := r.URL.Query().Get("format")
	switch format {
	case "zjson", "json":
		// XXX Should write appropriate ndjson content header.
		out = search.NewJSONOutput(w, search.DefaultMTU)
	case "bzng":
		// XXX Should write appropriate bzng content header.
		out = search.NewBzngOutput(w)
	default:
		http.Error(w, fmt.Sprintf("unsupported output format: %s", format), http.StatusBadRequest)
	}
	// XXX This always returns bad request but should return status codes
	// that reflect the nature of the returned error.
	if err := search.Search(r.Context(), req, out); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// PacketGetEndpoint returns the packets for a given conn_id as a
// pcap file.
func PacketSearchEndpoint(w http.ResponseWriter, r *http.Request) {
	spaceName, err := extractSpace(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	search := &api.PacketSearch{}
	if err := search.FromQuery(r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	preader, connid, err := pcap.Search(spaceName, *search)
	if err == pcap.ErrNoPacketsFound {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", connid))
	_, err = io.Copy(w, preader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func SpaceInfoEndpoint(w http.ResponseWriter, r *http.Request) {
	spacePath, err := extractSpace(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// XXX This is slow. Can easily cache result rather than scanning
	// whole file each time.
	info, err := space.Info(spacePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// extractSpace returns the unescaped space from the path of a request.
func extractSpace(r *http.Request) (string, error) {
	v := mux.Vars(r)
	space, ok := v["space"]
	if !ok {
		return "", errors.New("no space found")
	}
	return url.PathUnescape(space)
}
