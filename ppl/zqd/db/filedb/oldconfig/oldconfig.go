// Package oldconfig can read and migrate per-space json config files that
// were once used by zqd.
package oldconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/ppl/zqd/db/schema"
	"github.com/brimdata/zed/ppl/zqd/pcapstorage"
	"github.com/brimdata/zed/zqe"
	"go.uber.org/zap"
)

const (
	ConfigFile = "config.json"

	lastOldConfigVersion = 3
)

type ConfigV3 struct {
	Version int               `json:"version"`
	Name    string            `json:"name"`
	DataURI iosrc.URI         `json:"data_uri"`
	Storage api.StorageConfig `json:"storage"`
}

type ConfigV2 struct {
	Version  int               `json:"version"`
	Name     string            `json:"name"`
	DataURI  iosrc.URI         `json:"data_uri"`
	PcapPath string            `json:"pcap_path"`
	Storage  api.StorageConfig `json:"storage"`
}

type ConfigV1 struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	DataPath string `json:"data_path"`
	// XXX PcapPath should be named pcap_path in json land. To avoid having to
	// do a migration we'll keep this as-is for now.
	PcapPath string            `json:"packet_path"`
	Storage  api.StorageConfig `json:"storage"`
}

// versionCheck is used to establish the version of the loaded config file.
// This must always remain the same as the Version field in config.
type versionCheck struct {
	Version int `json:"version"`
}

type configMigrator struct {
	names map[string]api.SpaceID
}

func LoadConfigs(ctx context.Context, logger *zap.Logger, root iosrc.URI) (map[api.SpaceID]ConfigV3, error) {
	m := configMigrator{names: map[string]api.SpaceID{}}
	list, err := iosrc.ReadDir(ctx, root)
	if err != nil {
		return nil, err
	}
	res := make(map[api.SpaceID]ConfigV3)
	for _, l := range list {
		if !l.IsDir() {
			continue
		}
		dir := root.AppendPath(l.Name())
		config, err := m.loadConfig(ctx, dir)
		if err != nil {
			if zqe.IsNotFound(err) {
				logger.Debug("Config file not found", zap.String("uri", dir.String()))
			} else {
				logger.Warn("Error loading space", zap.String("uri", dir.String()), zap.Error(err))
			}
			continue
		}
		id := api.SpaceID(l.Name())
		m.names[config.Name] = id
		res[id] = config
	}
	return res, nil
}

// loadConfig loads the contents of config.json in a space's path.
func (m *configMigrator) loadConfig(ctx context.Context, spaceURI iosrc.URI) (ConfigV3, error) {
	var c ConfigV3
	p := spaceURI.AppendPath(ConfigFile)
	data, err := iosrc.ReadFile(ctx, p)
	if err != nil {
		return c, err
	}
	var vc versionCheck
	if err := json.Unmarshal(data, &vc); err != nil {
		return c, err
	}
	if vc.Version > lastOldConfigVersion {
		return c, fmt.Errorf("space config version %d ahead of binary version %d", vc.Version, lastOldConfigVersion)
	}
	if vc.Version < lastOldConfigVersion {
		return m.migrateConfig(vc.Version, data, spaceURI)
	}
	return c, json.Unmarshal(data, &c)
}

type migrator func([]byte, iosrc.URI) (int, []byte, error)

func (m *configMigrator) migrateConfig(version int, data []byte, spaceURI iosrc.URI) (ConfigV3, error) {
	var mg migrator
	for version < lastOldConfigVersion {
		switch version {
		case 0:
			mg = m.migrateConfigV1
		case 1:
			mg = migrateConfigV2
		case 2:
			mg = migrateConfigV3
		default:
			return ConfigV3{}, fmt.Errorf("unsupported config migration %d", version)
		}
		var err error
		if version, data, err = mg(data, spaceURI); err != nil {
			return ConfigV3{}, err
		}
	}
	var c ConfigV3
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, writeConfig(spaceURI, c)
}

func migrateConfigV3(data []byte, spaceuri iosrc.URI) (int, []byte, error) {
	var v2 ConfigV2
	if err := json.Unmarshal(data, &v2); err != nil {
		return 0, nil, err
	}
	if v2.PcapPath != "" {
		pcapuri, err := iosrc.ParseURI(v2.PcapPath)
		if err != nil {
			return 0, nil, err
		}
		du := v2.DataURI
		if du.IsZero() {
			du = spaceuri
		}
		if err := pcapstorage.MigrateV3(du, pcapuri); err != nil {
			return 0, nil, err
		}
	}
	c := ConfigV3{
		Version: 3,
		Name:    v2.Name,
		DataURI: v2.DataURI,
		Storage: v2.Storage,
	}
	d, err := json.Marshal(c)
	return 3, d, err
}

func migrateConfigV2(data []byte, _ iosrc.URI) (int, []byte, error) {
	var v1 ConfigV1
	if err := json.Unmarshal(data, &v1); err != nil {
		return 0, nil, err
	}
	if v1.DataPath == "." {
		v1.DataPath = ""
	}
	du, err := iosrc.ParseURI(v1.DataPath)
	if err != nil {
		return 0, nil, err
	}
	c := ConfigV2{
		Version:  2,
		Name:     v1.Name,
		DataURI:  du,
		PcapPath: v1.PcapPath,
		Storage:  v1.Storage,
	}
	d, err := json.Marshal(c)
	return 2, d, err
}

func (m *configMigrator) migrateConfigV1(data []byte, spaceURI iosrc.URI) (int, []byte, error) {
	var c ConfigV1
	if err := json.Unmarshal(data, &c); err != nil {
		return 0, nil, err
	}
	if c.Name == "" {
		// Ensure that name is not blank for spaces created before the
		// issue 721 work to use space ids.
		c.Name = path.Base(spaceURI.Path)
	}
	if _, ok := m.names[c.Name]; ok {
		c.Name = uniqueName(m.names, c.Name)
	}
	c.Name = schema.SafeSpaceName(c.Name)
	if c.Storage.Kind == api.UnknownStore {
		c.Storage.Kind = api.FileStore
	}
	d, err := json.Marshal(c)
	return 1, d, err
}

func writeConfig(spaceURI iosrc.URI, c ConfigV3) error {
	if c.Version != lastOldConfigVersion {
		return fmt.Errorf("writing an out of date config: expected version %d, got %d", lastOldConfigVersion, c.Version)
	}
	return iosrc.Replace(context.TODO(), spaceURI.AppendPath(ConfigFile), func(w io.Writer) error {
		return json.NewEncoder(w).Encode(c)
	})
}

func uniqueName(names map[string]api.SpaceID, proposed string) string {
	name := proposed
	for i := 1; ; i++ {
		if _, ok := names[name]; !ok {
			return name
		}
		name = fmt.Sprintf("%s_%02d", proposed, i)
	}
}
