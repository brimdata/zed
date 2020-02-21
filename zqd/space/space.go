package space

import (
	"os"
	"path/filepath"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/pcap"
)

func Info(path string) (*api.SpaceInfo, error) {
	bzngFile := filepath.Join(path, "all.bzng")
	info, err := os.Stat(bzngFile)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(bzngFile)
	if err != nil {
		return nil, err
	}
	reader, err := detector.LookupReader("bzng", f, resolver.NewContext())
	if err != nil {
		return nil, err
	}
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
		Name:          path,
		Size:          info.Size(),
		PacketSupport: pcap.HasPcaps(path),
	}
	if found {
		s.MinTime = &minTs
		s.MaxTime = &maxTs
	}
	return s, nil
}
