package zqd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqd/zeek"
	"github.com/gorilla/mux"
)

const pktIndexFile = "packets.idx.json"

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

func parseIP(host string) (net.IP, error) {
	ip := net.ParseIP(host)
	var err error
	if ip == nil {
		err = fmt.Errorf("invalid ip: %s", host)
	}
	return ip, err
}

func parseFlow(srcHost string, srcPort *uint16, dstHost string, dstPort *uint16) (pcap.Flow, error) {
	// convert ips
	src, err := parseIP(srcHost)
	if err != nil {
		return pcap.Flow{}, err
	}
	dst, err := parseIP(dstHost)
	if dst == nil {
		return pcap.Flow{}, err
	}
	if srcPort == nil || dstPort == nil {
		return pcap.Flow{}, fmt.Errorf("port(s) missing in pcap request")
	}
	return pcap.NewFlow(src, int(*srcPort), dst, int(*dstPort)), nil
}

func handlePacketSearch(root string, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(root, w, r)
	if s == nil {
		return
	}
	req := &api.PacketSearch{}
	if err := req.FromQuery(r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if s.PacketPath() == "" || !s.HasFile(pktIndexFile) {
		http.Error(w, "space has no pcaps", http.StatusNotFound)
		return
	}
	index, err := pcap.LoadIndex(s.DataPath(pktIndexFile))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.Open(s.PacketPath())
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
	var search *pcap.Search
	switch req.Proto {
	default:
		msg := fmt.Sprintf("unsupported proto type: %s", req.Proto)
		http.Error(w, msg, http.StatusBadRequest)
		return
	case "tcp":
		flow, err := parseFlow(req.SrcHost, req.SrcPort, req.DstHost, req.DstPort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		search = pcap.NewTCPSearch(req.Span, flow)
	case "udp":
		flow, err := parseFlow(req.SrcHost, req.SrcPort, req.DstHost, req.DstPort)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		search = pcap.NewUDPSearch(req.Span, flow)
	case "icmp":
		src, err := parseIP(req.SrcHost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		dst, err := parseIP(req.DstHost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		search = pcap.NewICMPSearch(req.Span, src, dst)
	}
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", search.ID()))
	err = search.Run(w, slicer)
	if err != nil {
		if err == pcap.ErrNoPacketsFound {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
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
		if err != nil {
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
	s := extractSpace(root, w, r)
	if s == nil {
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
		Name:          s.Name(),
		Size:          stat.Size(),
		PacketSupport: s.HasFile(pktIndexFile),
		PacketPath:    s.PacketPath(),
	}
	if found {
		info.MinTime = &minTs
		info.MaxTime = &maxTs
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleSpacePost(root string, w http.ResponseWriter, r *http.Request) {
	var req api.SpacePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := space.Create(root, req.Name, req.DataDir)
	if err != nil {
		status := http.StatusInternalServerError
		if err == space.ErrSpaceExists {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	res := api.SpacePostResponse{
		Name:    s.Name(),
		DataDir: s.DataPath(),
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		// XXX Add zap here.
		log.Println("Error writing response", err)
	}
}

func handlePacketPost(root string, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(root, w, r)
	if s == nil {
		return
	}
	var req api.PacketPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pcapfile, err := os.Open(req.Path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	defer pcapfile.Close()
	logdir := s.DataPath(".tmp.zeeklogs")
	if err := os.Mkdir(logdir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(logdir)
	// create logs
	logfiles, err := zeek.Logify(r.Context(), logdir, pcapfile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// convert logs into sorted bzng
	zr, err := scanner.OpenFiles(resolver.NewContext(), logfiles...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer zr.Close()
	// For the time being, this endpoint will overwrite any underlying data.
	// In order to get rid errors on any concurrent searches on this space,
	// write bzng to a temp file and rename on successful conversion.
	bzngfile, err := s.CreateFile("all.bzng.tmp")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	zw := bzngio.NewWriter(bzngfile)
	const program = "_path != packet_filter _path != loaded_scripts | sort -limit 1000000000 -r ts"
	if err := search.Copy(r.Context(), zw, zr, program); err != nil {
		// If an error occurs here close and remove tmp bzngfile, lest we start
		// leaking files and file descriptors.
		bzngfile.Close()
		os.Remove(bzngfile.Name())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := bzngfile.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.Rename(bzngfile.Name(), s.DataPath("all.bzng")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// create pcap index
	if _, err := pcapfile.Seek(0, 0); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	idx, err := pcap.CreateIndex(pcapfile, 10000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pcapIdx, err := s.CreateFile(pktIndexFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer pcapIdx.Close()
	if err := json.NewEncoder(pcapIdx).Encode(idx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// update space config
	if err := s.SetPacketPath(req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func extractSpace(root string, w http.ResponseWriter, r *http.Request) *space.Space {
	name := extractSpaceName(w, r)
	if name == "" {
		return nil
	}
	s, err := space.Open(root, name)
	if err != nil {
		if err == space.ErrSpaceNotExist {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return nil
	}
	return s
}

// extractSpaceName returns the unescaped space from the path of a request.
func extractSpaceName(w http.ResponseWriter, r *http.Request) string {
	v := mux.Vars(r)
	space, ok := v["space"]
	if !ok {
		http.Error(w, "no space name in path", http.StatusBadRequest)
		return ""
	}
	return space
}
