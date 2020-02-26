package zqd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/gorilla/mux"
)

const pktIndexFile = "packets.idx.json"
const pcapFile = "packets.pcap"

func handleSearch(root string, w http.ResponseWriter, r *http.Request) {
	var req api.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := space.Open(root, req.Space)
	if err != nil {
		status := http.StatusInternalServerError
		if err == space.ErrSpaceNotExist {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
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
	if err := search.Search(r.Context(), s, req, out); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func handlePacketSearch(root string, w http.ResponseWriter, r *http.Request) {
	spaceName, err := extractSpace(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := space.Open(root, spaceName)
	if err != nil {
		status := http.StatusInternalServerError
		if err == space.ErrSpaceNotExist {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	req := &api.PacketSearch{}
	if err := req.FromQuery(r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if !s.HasFile(pcapFile) || !s.HasFile(pktIndexFile) {
		http.Error(w, "space has no pcaps", http.StatusNotFound)
		return
	}
	index, err := pcap.LoadIndex(s.DataPath(pktIndexFile))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// XXX pcapFile should be able to live outside space data path?
	f, err := s.OpenFile(pcapFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	slicer, err := pcap.NewSlicer(f, index, req.Span)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	search, err := pcap.NewSearch(
		req.Span,
		req.Proto,
		req.SrcHost,
		req.SrcPort,
		req.DstHost,
		req.DstPort,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", search.ID()))
	err = search.Run(w, slicer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func handleSpaceList(root string, w http.ResponseWriter, r *http.Request) {
	info, err := ioutil.ReadDir(root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	spaces := []string{}
	for _, subdir := range info {
		if !subdir.IsDir() {
			continue
		}
		s, err := space.Open(root, subdir.Name())
		if err != nil || s.HasFile("all.bzng") {
			continue
		}
		spaces = append(spaces, s.Name())
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(spaces); err != nil {
		// XXX Add zap here.
		log.Println("Error writing response", err)
	}
}

func handleSpaceGet(root string, w http.ResponseWriter, r *http.Request) {
	spaceName, err := extractSpace(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := space.Open(root, spaceName)
	if err != nil {
		if err == space.ErrSpaceNotExist {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	f, err := s.OpenFile("all.bzng")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer f.Close()
	// XXX This is slow. Can easily cache result rather than scanning
	// whole file each time.
	reader, err := detector.LookupReader("bzng", f, resolver.NewContext())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	minTs := nano.MaxTs
	maxTs := nano.MinTs
	var found bool
	for {
		rec, err := reader.Read()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	info := &api.SpaceInfo{
		Name:          spaceName,
		Size:          stat.Size(),
		PacketSupport: s.HasFile(pktIndexFile),
	}
	if found {
		info.MinTime = &minTs
		info.MaxTime = &maxTs
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
	return space, nil
}
