package zqd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pcap/pcapio"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/ingest"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func handleSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := space.Open(c.Root, req.Space)
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
		return
	}
	// XXX This always returns bad request but should return status codes
	// that reflect the nature of the returned error.
	w.Header().Set("Content-Type", "application/ndjson")
	if err := search.Search(r.Context(), s, req, out); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func handlePacketSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	req := &api.PacketSearch{}
	if err := req.FromQuery(r.URL.Query()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	if s.PacketPath() == "" || !s.HasFile(ingest.PcapIndexFile) {
		http.Error(w, "space has no pcaps", http.StatusNotFound)
		return
	}
	index, err := pcap.LoadIndex(s.DataPath(ingest.PcapIndexFile))
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pcapReader, err := pcapio.NewReader(slicer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var search *pcap.Search
	switch req.Proto {
	default:
		msg := fmt.Sprintf("unsupported proto type: %s", req.Proto)
		http.Error(w, msg, http.StatusBadRequest)
		return
	case "tcp":
		flow := pcap.NewFlow(req.SrcHost, int(req.SrcPort), req.DstHost, int(req.DstPort))
		search = pcap.NewTCPSearch(req.Span, flow)
	case "udp":
		flow := pcap.NewFlow(req.SrcHost, int(req.SrcPort), req.DstHost, int(req.DstPort))
		search = pcap.NewUDPSearch(req.Span, flow)
	case "icmp":
		search = pcap.NewICMPSearch(req.Span, req.SrcHost, req.DstHost)
	}
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", search.ID()))
	err = search.Run(w, pcapReader)
	if err != nil {
		if err == pcap.ErrNoPacketsFound {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
}

func handleSpaceList(c *Core, w http.ResponseWriter, r *http.Request) {
	info, err := ioutil.ReadDir(c.Root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	spaces := []string{}
	for _, subdir := range info {
		if !subdir.IsDir() {
			continue
		}
		s, err := space.Open(c.Root, subdir.Name())
		if err != nil {
			continue
		}
		spaces = append(spaces, s.Name())
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(spaces); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func handleSpaceGet(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	info := &api.SpaceInfo{
		Name:          s.Name(),
		PacketSupport: s.HasFile(ingest.PcapIndexFile),
		PacketPath:    s.PacketPath(),
	}
	if s.HasFile("all.bzng") {
		f, err := s.OpenFile("all.bzng")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info.Size = stat.Size()

		minTs, maxTs, err := s.GetTimes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		info.MinTime = minTs
		info.MaxTime = maxTs
	}
	if info.PacketPath != "" {
		if pstat, err := os.Stat(info.PacketPath); err == nil {
			info.PacketSize = pstat.Size()
		} else if !os.IsNotExist(err) {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func handleSpacePost(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.SpacePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s, err := space.Create(c.Root, req.Name, req.DataDir)
	if err != nil {
		status := http.StatusInternalServerError
		if err == space.ErrSpaceExists {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}
	res := api.SpacePostResponse{
		Name:    s.Name(),
		DataDir: s.DataPath(),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func handleSpaceDelete(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	if err := s.Delete(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handlePacketPost(c *Core, w http.ResponseWriter, r *http.Request) {
	if !c.HasZeek() {
		http.Error(w, "packet post not supported: zeek not found", http.StatusInternalServerError)
		return
	}
	logger := c.requestLogger(r)
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	var req api.PacketPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	proc, err := ingest.Pcap(r.Context(), s, req.Path, c.ZeekLauncher, c.SortLimit)
	if err != nil {
		if errors.Is(err, pcapio.ErrCorruptPcap) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/ndjson")
	w.WriteHeader(http.StatusAccepted)
	pipe := api.NewJSONPipe(w)
	taskID := c.getTaskID()
	taskStart := api.TaskStart{Type: "TaskStart", TaskID: taskID}
	if err = pipe.Send(taskStart); err != nil {
		logger.Warn("Error sending payload", zap.Error(err))
		return
	}
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for {
		var done bool
		select {
		case <-proc.Done():
			done = true
		case <-proc.Snap():
		case <-ticker.C:
		}

		var minTs, maxTs *nano.Ts
		if minTs, maxTs, err = s.GetTimes(); err != nil {
			break
		}
		status := api.PacketPostStatus{
			Type:           "PacketPostStatus",
			StartTime:      proc.StartTime,
			UpdateTime:     nano.Now(),
			PacketSize:     proc.PcapSize,
			PacketReadSize: proc.PcapReadSize(),
			SnapshotCount:  proc.SnapshotCount(),
			MinTime:        minTs,
			MaxTime:        maxTs,
		}
		if err := pipe.Send(status); err != nil {
			logger.Warn("Error sending payload", zap.Error(err))
			return
		}
		if done {
			break
		}
	}
	taskEnd := api.TaskEnd{Type: "TaskEnd", TaskID: taskID}
	if err := proc.Err(); err != nil {
		var ok bool
		taskEnd.Error, ok = err.(*api.Error)
		if !ok {
			taskEnd.Error = &api.Error{Type: "Error", Message: err.Error()}
		}
	}
	if err = pipe.SendFinal(taskEnd); err != nil {
		logger.Warn("Error sending payload", zap.Error(err))
		return
	}
}

func extractSpace(c *Core, w http.ResponseWriter, r *http.Request) *space.Space {
	name := extractSpaceName(w, r)
	if name == "" {
		return nil
	}
	s, err := space.Open(c.Root, name)
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
