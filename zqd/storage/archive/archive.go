package archivestore

import (
	"context"
	"os"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/storage"
)

func Load(path string) (*ArkStorage, error) {
	ark, err := archive.OpenArchive(path)
	if err != nil {
		return nil, err
	}
	return &ArkStorage{ark: ark}, nil
}

type ArkStorage struct {
	ark *archive.Archive
}

func (as *ArkStorage) NativeDirection() zbuf.Direction {
	return zbuf.Direction(as.ark.Meta.DataSortForward)
}

func (as *ArkStorage) Open(ctx context.Context, span nano.Span) (zbuf.ReadCloser, error) {
	var (
		err     error
		readers []zbuf.Reader
	)
	defer func() {
		if err != nil {
			for _, r := range readers {
				zf := r.(*zbuf.File)
				zf.Close()
			}
		}
	}()
	zctx := resolver.NewContext()
	err = archive.SpanWalk(as.ark, func(sp nano.Span, zngpath string) error {
		if !span.Overlaps(sp) {
			return nil
		}
		f, err := fs.Open(zngpath)
		if err != nil {
			return err
		}
		r := zngio.NewReader(f, zctx)
		readers = append(readers, zbuf.NewFile(r, f, f.Name()))
		return nil
	})
	if err != nil {
		return nil, err
	}
	combiner := zbuf.NewCombiner(readers, as.NativeDirection())
	return combiner, nil
}

func (as *ArkStorage) Summary(_ context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = storage.KindArchive
	return sum, archive.SpanWalk(as.ark, func(sp nano.Span, zngpath string) error {
		sinfo, err := os.Stat(zngpath)
		if err != nil {
			return err
		}
		sum.DataBytes += sinfo.Size()
		if sum.Span.Dur == 0 {
			sum.Span = sp
		} else {
			sum.Span = sum.Span.Union(sp)
		}
		return nil
	})
}
