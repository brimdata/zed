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

func Load(path string) (*Storage, error) {
	ark, err := archive.OpenArchive(path)
	if err != nil {
		return nil, err
	}
	return &Storage{ark: ark}, nil
}

type Storage struct {
	ark *archive.Archive
}

func (s *Storage) NativeDirection() zbuf.Direction {
	return zbuf.Direction(s.ark.Meta.DataSortForward)
}

func (s *Storage) Open(ctx context.Context, span nano.Span) (zbuf.ReadCloser, error) {
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
	err = archive.SpanWalk(s.ark, func(sp nano.Span, zngpath string) error {
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
	combiner := zbuf.NewCombiner(readers, s.NativeDirection())
	return combiner, nil
}

func (s *Storage) Summary(_ context.Context) (storage.Summary, error) {
	var sum storage.Summary
	sum.Kind = storage.ArchiveStore
	return sum, archive.SpanWalk(s.ark, func(sp nano.Span, zngpath string) error {
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
