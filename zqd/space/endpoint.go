package space

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/pcap"
)

func HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	root := "."
	info, err := ioutil.ReadDir(root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var spaces []string
	for _, subdir := range info {
		if !subdir.IsDir() {
			continue
		}
		dataFile := filepath.Join(root, subdir.Name(), "all.bzng")
		s, err := os.Stat(dataFile)
		if err != nil || s.IsDir() {
			continue
		}
		spaces = append(spaces, subdir.Name())
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spaces)
}

func spaceInfo(spaceName, path string) (*api.SpaceInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader := detector.LookupReader("bzng", f, resolver.NewContext())
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
		Name:          spaceName,
		Size:          info.Size(),
		PacketSupport: pcap.HasPcaps(spaceName),
	}
	if found {
		s.MinTime = &minTs
		s.MaxTime = &maxTs
	}
	return s, nil
}

func parseSpace(path string) (string, string) {
	//XXX need to sanitize this path
	spaceName := strings.Replace(path, "/space/", "", 1)
	if spaceName == "" {
		return "", ""
	}
	if strings.HasSuffix(spaceName, "/packet") {
		return "", spaceName[:len(spaceName)-7]
	}
	return spaceName, ""
}

func HandleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	spaceName, pcapSpace := parseSpace(r.URL.Path)
	if pcapSpace != "" {
		pcap.HandleGet(w, r, pcapSpace)
		return
	}
	root := "."
	path := filepath.Join(root, spaceName, "all.bzng")
	// XXX this is slow.  can easily cache result rather than scanning
	// whole file each time.
	info, err := spaceInfo(spaceName, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
