package transform

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var SortMemMaxBytes int64 = 128 * 1024 * 1024

type SortReader struct {
	peeker  *zbuf.Peeker
	maxSize int64
	baseDir string
	tmpDir  string
	sortFn  expr.SortFn
	ctx     context.Context

	once        sync.Once
	ch          chan result
	records     []*zng.Record
	currentSize int64
}

func NewSortReader(ctx context.Context, r zbuf.Reader, maxSize int64, sortFn expr.SortFn, dir string) *SortReader {
	if maxSize == 0 {
		maxSize = SortMemMaxBytes
	}
	return &SortReader{
		ctx:     ctx,
		peeker:  zbuf.NewPeeker(r),
		maxSize: maxSize,
		baseDir: dir,
		sortFn:  sortFn,
		ch:      make(chan result),
	}
}

type result struct {
	record *zng.Record
	err    error
}

func (s *SortReader) run() {
	var err error
	var batch zbuf.Batch
	defer s.cleanup()
	for s.ctx.Err() == nil {
		batch, err = zbuf.ReadBatchSize(s.peeker, s.maxSize)
		if err != nil {
			s.ch <- result{nil, err}
		}
		records := batch.Records()
		expr.SortStable(records, s.sortFn)
		batch = zbuf.NewArray(records, batch.Span())
		// if we are at EOF break loop and finish
		if rec, _ := s.peeker.Peek(); rec == nil {
			break
		}
		if err = s.flushBatch(batch); err != nil {
			s.ch <- result{nil, err}
			return
		}
	}
	s.finish(batch)
}

func (s *SortReader) flushBatch(batch zbuf.Batch) error {
	s.createTmpDir()
	f, err := ioutil.TempFile(s.tmpDir, ".tmp-*.zng")
	if err != nil {
		return err
	}
	defer f.Close()
	r := zbuf.NewBatchReader(batch)
	bw := bufio.NewWriter(f)
	w := zngio.NewWriter(bw, zio.WriterFlags{})
	if err := zbuf.CopyWithContext(s.ctx, w, r); err != nil {
		return err
	}
	return bw.Flush()
}

func (s *SortReader) finish(batch zbuf.Batch) error {
	br := zbuf.NewBatchReader(batch)
	if s.tmpDir == "" {
		s.send(br)
		return nil
	}
	zctx := resolver.NewContext()
	readers := []zbuf.Reader{br}
	files, err := filepath.Glob(filepath.Join(s.tmpDir, "*.zng"))
	if err != nil {
		return err
	}
	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		zr := zbuf.NewPeeker(zngio.NewReader(bufio.NewReader(f), zctx))
		readers = append(readers, zr)
	}
	combiner := zbuf.NewCombiner(readers, s.sortFn)
	s.send(combiner)
	return combiner.Close()
}

func (s *SortReader) send(r zbuf.Reader) {
	for s.ctx.Err() == nil {
		rec, err := r.Read()
		// XXX may need ctx interrupt here?
		s.ch <- result{rec, err}
		if rec == nil || err != nil {
			break
		}
	}
}

func (s *SortReader) cleanup() error {
	if s.tmpDir != "" {
		tmp := s.tmpDir
		s.tmpDir = ""
		return os.RemoveAll(tmp)
	}
	return nil
}

func (s *SortReader) createTmpDir() error {
	var err error
	if s.tmpDir == "" {
		s.tmpDir, err = ioutil.TempDir(s.baseDir, "SortReader")
	}
	return err
}

func (s *SortReader) Read() (*zng.Record, error) {
	s.once.Do(func() { go s.run() })
	select {
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case res := <-s.ch:
		return res.record, res.err
	}
}
