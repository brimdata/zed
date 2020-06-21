package space

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqe"
)

var nameRegexp = regexp.MustCompile("[^-.a-zA-Z0-9_]")

const configVersion = 1

type config struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	DataPath string `json:"data_path"`
	// XXX PcapPath should be named pcap_path in json land. To avoid having to
	// do a migration we'll keep this as-is for now.
	PcapPath  string           `json:"packet_path"`
	Storage   storage.Config   `json:"storage"`
	Subspaces []subspaceConfig `json:"subspaces"`
}

type subspaceConfig struct {
	ID          api.SpaceID                `json:"id"`
	Name        string                     `json:"name"`
	OpenOptions storage.ArchiveOpenOptions `json:"open_options"`
}

func (c config) clone() config {
	n := c
	n.Subspaces = append([]subspaceConfig{}, c.Subspaces...)
	return n
}

func (c config) subspaceIndex(id api.SpaceID) int {
	for i, sub := range c.Subspaces {
		if sub.ID == id {
			return i
		}
	}
	return -1
}

// loadConfig loads the contents of config.json in a space's path.
func loadConfig(spacePath string) (config, error) {
	var c config
	path := filepath.Join(spacePath, configFile)
	if err := fs.UnmarshalJSONFile(path, &c); err != nil {
		return c, err
	}
	if c.Name == "" {
		// Ensure that name is not blank for spaces created before the
		// zq#721 work to use space ids.
		c.Name = filepath.Base(spacePath)
	}
	if c.Storage.Kind == storage.UnknownStore {
		c.Storage.Kind = storage.FileStore
	}
	return c, nil
}

func writeConfig(spacePath string, c config) error {
	path := filepath.Join(spacePath, configFile)
	return fs.MarshalJSONFile(c, path, 0644)
}

func validateName(names map[string]api.SpaceID, name string) error {
	if name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}
	if nameRegexp.MatchString(name) {
		return zqe.E(zqe.Invalid, "invalid space name")
	}
	if _, ok := names[name]; ok {
		return zqe.E(zqe.Conflict, "space with name '%s' already exists", name)
	}
	return nil
}

// safeName converts the proposed name to a name that adheres to the constraints
// placed on a space's name (i.e. follows the name regex and is unique). In
// order to ensure the generated name is unique this should be called with the
// manager lock held.
func safeName(names map[string]api.SpaceID, proposed string) string {
	base := filepath.Base(proposed)
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.ReplaceAll(base, ".", "_")
	base = strings.ReplaceAll(base, "-", "_")
	base = nameRegexp.ReplaceAllString(base, "")
	name := base
	// ensure uniqueness
	for i := 1; ; i++ {
		if _, ok := names[name]; !ok {
			return name
		}
		name = fmt.Sprintf("%s_%02d", base, i)
	}
}
