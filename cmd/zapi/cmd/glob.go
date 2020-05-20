package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/brimsec/zq/pkg/glob"
	"github.com/brimsec/zq/zqd/api"
)

var ErrNoMatch = errors.New("no match")

func SpaceGlob(ctx context.Context, client *API, patterns ...string) ([]api.SpaceInfo, error) {
	all, err := client.SpaceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch spaces: %w", err)
	}
	if len(all) == 0 {
		return nil, errors.New("no spaces exist")
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
