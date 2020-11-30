package index

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/segmentio/ksuid"
)

var indexFileRegex = regexp.MustCompile(`idx-([0-9A-Za-z]{27}).zng$`)

func Find(ctx context.Context, u iosrc.URI, patterns ...string) (*zng.Record, error) {
	finder, err := microindex.NewFinder(ctx, resolver.NewContext(), u)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", u, err)
	}
	defer finder.Close()
	keys, err := finder.ParseKeys(patterns...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", finder.Path(), err)
	}
	rec, err := finder.Lookup(keys)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", finder.Path(), err)
	}
	return rec, nil
}

func FindAll(ctx context.Context, u iosrc.URI, patterns ...string) (zbuf.Batch, error) {
	finder, err := microindex.NewFinder(ctx, resolver.NewContext(), u)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", u, err)
	}
	defer finder.Close()
	keys, err := finder.ParseKeys(patterns...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", finder.Path(), err)
	}
	batch, err := finder.LookupBatch(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", finder.Path(), err)
	}
	return batch, nil
}

func Printall(u iosrc.URI) error {
	r, err := microindex.NewReader(resolver.NewContext(), u.String())
	if err != nil {
		return err
	}
	sr, err := r.NewSectionReader(0)
	if err != nil {
		return err
	}
	w := tzngio.NewWriter(zio.NopCloser(os.Stdout))
	return zbuf.Copy(w, sr)
}

func IndexPath(dir iosrc.URI, id ksuid.KSUID) iosrc.URI {
	return dir.AppendPath(fmt.Sprintf("idx-%s.zng", id))
}

func parseIndexFile(name string) (ksuid.KSUID, error) {
	match := indexFileRegex.FindStringSubmatch(name)
	if match == nil {
		return ksuid.Nil, fmt.Errorf("invalid index file: %s", name)
	}
	return ksuid.Parse(match[1])
}
