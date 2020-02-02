package space

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mccanne/zq/zio/detector"
	"github.com/mccanne/zq/zng"
	"github.com/mccanne/zq/zng/resolver"
	"github.com/mccanne/zq/zqd/api"
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
	var first, last *zng.Record
	for {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		if first == nil {
			first = rec
		}
		last = rec
	}
	if first == nil {
		return nil, errors.New("empty space") //XXX
	}
	return &api.SpaceInfo{
		Name:    spaceName,
		MinTime: &first.Ts,
		MaxTime: &last.Ts,
		Size:    info.Size(),
	}, nil
}

func HandleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "bad method", http.StatusBadRequest)
		return
	}
	//XXX need to sanitize spaceName
	spaceName := strings.Replace(r.URL.Path, "/space/", "", 1)
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
