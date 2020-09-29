package space

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"unicode"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/ppl/zqd/pcapstorage"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/azngio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
)

const configVersion = 4

func invalidSpaceNameRune(r rune) bool {
	return r == '/' || !unicode.IsPrint(r)
}

func validSpaceName(s string) bool {
	return strings.IndexFunc(s, invalidSpaceNameRune) == -1
}

type config struct {
	Version   int               `json:"version"`
	Name      string            `json:"name"`
	DataURI   iosrc.URI         `json:"data_uri"`
	Storage   api.StorageConfig `json:"storage"`
	Subspaces []subspaceConfig  `json:"subspaces"`
}

type configV2 struct {
	Version   int               `json:"version"`
	Name      string            `json:"name"`
	DataURI   iosrc.URI         `json:"data_uri"`
	PcapPath  string            `json:"pcap_path"`
	Storage   api.StorageConfig `json:"storage"`
	Subspaces []subspaceConfig  `json:"subspaces"`
}

type configV1 struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	DataPath string `json:"data_path"`
	// XXX PcapPath should be named pcap_path in json land. To avoid having to
	// do a migration we'll keep this as-is for now.
	PcapPath  string            `json:"packet_path"`
	Storage   api.StorageConfig `json:"storage"`
	Subspaces []subspaceConfig  `json:"subspaces"`
}

// versionCheck is used to establish the version of the loaded config file.
// This must always remain the same as the Version field in config.
type versionCheck struct {
	Version int `json:"version"`
}

type subspaceConfig struct {
	ID          api.SpaceID            `json:"id"`
	Name        string                 `json:"name"`
	OpenOptions api.ArchiveOpenOptions `json:"open_options"`
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
func (m *Manager) loadConfig(ctx context.Context, spaceURI iosrc.URI) (config, error) {
	var c config
	p := spaceURI.AppendPath(configFile)
	r, err := iosrc.NewReader(ctx, p)
	if err != nil {
		return c, err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return c, err
	}
	if err := r.Close(); err != nil {
		return c, err
	}
	var vc versionCheck
	if err := json.Unmarshal(data, &vc); err != nil {
		return c, err
	}
	if vc.Version > configVersion {
		return c, fmt.Errorf("space config version %d ahead of binary version %d", vc.Version, configVersion)
	}
	if vc.Version < configVersion {
		return m.migrateConfig(ctx, vc.Version, data, spaceURI)
	}
	return c, json.Unmarshal(data, &c)
}

type migrator func(context.Context, []byte, iosrc.URI) (int, []byte, error)

func (m *Manager) migrateConfig(ctx context.Context, version int, data []byte, spaceURI iosrc.URI) (config, error) {
	var mg migrator
	for version < configVersion {
		switch version {
		case 0:
			mg = m.migrateConfigV1
		case 1:
			mg = migrateConfigV2
		case 2:
			mg = migrateConfigV3
		case 3:
			mg = migrateConfigV4
		default:
			return config{}, fmt.Errorf("unsupported config migration %d", version)
		}
		var err error
		if version, data, err = mg(ctx, data, spaceURI); err != nil {
			return config{}, err
		}
	}
	var c config
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, writeConfig(spaceURI, c)
}

func migrateAlphaAllZng(ctx context.Context, u iosrc.URI) error {
	return iosrc.Replace(ctx, u, func(w io.Writer) error {
		zctx := resolver.NewContext()
		zw := zngio.NewWriter(bufwriter.New(zio.NopCloser(w)), zngio.WriterOpts{
			StreamRecordsMax: zngio.DefaultStreamRecordsMax,
			LZ4BlockSize:     zngio.DefaultLZ4BlockSize,
		})
		rc, err := iosrc.NewReader(context.Background(), u)
		if err != nil {
			return err
		}
		ar, err := azngio.NewReader(rc, zctx)
		if err != nil {
			rc.Close()
			return err
		}
		err = zbuf.Copy(zw, ar)
		if rcErr := rc.Close(); err == nil {
			err = rcErr
		}
		if zwErr := zw.Close(); err == nil {
			err = zwErr
		}
		return err
	})
}

func findV3File(ctx context.Context, spaceuri iosrc.URI, v3 *config) (iosrc.URI, bool, error) {
	for _, f := range []string{"all.zng", "all.bzng"} {
		var uri iosrc.URI
		if v3.DataURI.IsZero() {
			uri = spaceuri.AppendPath(f)
		} else {
			uri = v3.DataURI.AppendPath(f)
		}
		exists, err := iosrc.Exists(ctx, uri)
		if err != nil {
			return iosrc.URI{}, false, err
		}
		if exists {
			return uri, true, nil
		}
	}
	return iosrc.URI{}, false, nil
}

func migrateConfigV4(ctx context.Context, data []byte, spaceuri iosrc.URI) (int, []byte, error) {
	var v3 config
	if err := json.Unmarshal(data, &v3); err != nil {
		return 0, nil, err
	}
	if v3.Storage.Kind == api.FileStore {
		allZng, exists, err := findV3File(ctx, spaceuri, &v3)
		if err != nil {
			return 0, nil, err
		}
		if exists {
			if err := migrateAlphaAllZng(ctx, allZng); err != nil {
				return 0, nil, err
			}
		}
	}
	c := v3
	c.Version = 4
	d, err := json.Marshal(c)
	return 4, d, err
}

func migrateConfigV3(ctx context.Context, data []byte, spaceuri iosrc.URI) (int, []byte, error) {
	var v2 configV2
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
		if err := pcapstorage.MigrateV3(ctx, du, pcapuri); err != nil {
			return 0, nil, err
		}
	}
	c := config{
		Version:   3,
		Name:      v2.Name,
		DataURI:   v2.DataURI,
		Storage:   v2.Storage,
		Subspaces: v2.Subspaces,
	}
	d, err := json.Marshal(c)
	return 3, d, err
}

func migrateConfigV2(_ context.Context, data []byte, _ iosrc.URI) (int, []byte, error) {
	var v1 configV1
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
	c := configV2{
		Version:   2,
		Name:      v1.Name,
		DataURI:   du,
		PcapPath:  v1.PcapPath,
		Storage:   v1.Storage,
		Subspaces: v1.Subspaces,
	}
	d, err := json.Marshal(c)
	return 2, d, err
}

func (m *Manager) migrateConfigV1(_ context.Context, data []byte, spaceURI iosrc.URI) (int, []byte, error) {
	var c configV1
	if err := json.Unmarshal(data, &c); err != nil {
		return 0, nil, err
	}
	if c.Name == "" {
		// Ensure that name is not blank for spaces created before the
		// zq#721 work to use space ids.
		c.Name = path.Base(spaceURI.Path)
	}
	if _, ok := m.names[c.Name]; ok {
		c.Name = uniqueName(m.names, c.Name)
	}
	c.Name = safeName(c.Name)
	if c.Storage.Kind == api.UnknownStore {
		c.Storage.Kind = api.FileStore
	}
	d, err := json.Marshal(c)
	return 1, d, err
}

func writeConfig(spaceURI iosrc.URI, c config) error {
	if c.Version != configVersion {
		return fmt.Errorf("writing an out of date config: expected version %d, got %d", configVersion, c.Version)
	}
	return iosrc.Replace(context.TODO(), spaceURI.AppendPath(configFile), func(w io.Writer) error {
		return json.NewEncoder(w).Encode(c)
	})
}

func validateName(names map[string]api.SpaceID, name string) error {
	if name == "" {
		return zqe.E(zqe.Invalid, "cannot set name to an empty string")
	}
	if !validSpaceName(name) {
		return zqe.E(zqe.Invalid, "name may not contain '/' or non-printable characters")
	}
	if _, ok := names[name]; ok {
		return zqe.E(zqe.Conflict, "space with name '%s' already exists", name)
	}
	return nil
}

// safeName converts the proposed name to a name that adheres to the constraints
// placed on a space's name (i.e. follows the name regex).
func safeName(proposed string) string {
	var sb strings.Builder
	for _, r := range proposed {
		if invalidSpaceNameRune(r) {
			r = '_'
		}
		sb.WriteRune(r)
	}
	return sb.String()
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
