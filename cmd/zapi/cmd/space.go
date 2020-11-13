package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/api/client"
	"github.com/brimsec/zq/pkg/glob"
	"github.com/brimsec/zq/pkg/units"
	"github.com/brimsec/zq/ppl/archive"
)

var (
	ErrNoMatch       = errors.New("no match")
	ErrNoSpacesExist = errors.New("no spaces exist")
)

type SpaceCreateFlags struct {
	kind     api.StorageKind
	datapath string
	thresh   units.Bytes
}

func (f *SpaceCreateFlags) SetFlags(fs *flag.FlagSet) {
	f.kind = api.FileStore
	f.thresh = archive.DefaultLogSizeThreshold
	fs.Var(&f.kind, "k", "kind of storage for this space")
	fs.StringVar(&f.datapath, "d", "", "specific directory for storage data")
	fs.Var(&f.thresh, "thresh", "target size of chopped files, as '10MB', '4GiB', etc.")
}

func (f *SpaceCreateFlags) Init() error {
	return nil
}

func (f *SpaceCreateFlags) Create(ctx context.Context, conn *client.Connection, name string) (*api.SpaceInfo, error) {
	req := api.SpacePostRequest{
		Name:     name,
		DataPath: f.datapath,
		Storage: &api.StorageConfig{
			Kind: f.kind,
			Archive: &api.ArchiveConfig{
				CreateOptions: &api.ArchiveCreateOptions{
					LogSizeThreshold: (*int64)(&f.thresh),
				},
			},
		},
	}
	return conn.SpacePost(ctx, req)
}

func SpaceGlob(ctx context.Context, conn *client.Connection, patterns ...string) ([]api.SpaceInfo, error) {
	all, err := conn.SpaceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch spaces: %w", err)
	}
	if len(all) == 0 {
		return nil, ErrNoSpacesExist
	}
	var spaces []api.SpaceInfo
	if len(patterns) == 0 {
		spaces = all
	} else {
		m := newSpacemap(all)
		names, err := glob.Globv(patterns, m.names())
		if err != nil {
			return nil, err
		}
		spaces = m.matches(names)
		if len(spaces) == 0 {
			return nil, ErrNoMatch
		}
	}
	sort.Slice(spaces, func(i, j int) bool {
		return spaces[i].Name < spaces[j].Name
	})
	return spaces, nil
}

func GetSpaceID(ctx context.Context, conn *client.Connection, name string) (api.SpaceID, error) {
	spaces, err := SpaceGlob(ctx, conn, name)
	if err != nil {
		return "", err
	}
	if len(spaces) > 1 {
		list := strings.Join(api.SpaceInfos(spaces).Names(), ", ")
		return "", fmt.Errorf("found multiple matching spaces: %s", list)
	}
	return spaces[0].ID, nil
}

type spacemap map[string]api.SpaceInfo

func newSpacemap(infos []api.SpaceInfo) spacemap {
	m := make(spacemap)
	for _, info := range infos {
		m[info.Name] = info
	}
	return m
}

func (s spacemap) names() (names []string) {
	for key := range s {
		names = append(names, key)
	}
	return
}

func (s spacemap) matches(names []string) []api.SpaceInfo {
	infos := make([]api.SpaceInfo, len(names))
	for i, name := range names {
		infos[i] = s[name]
	}
	return infos
}
