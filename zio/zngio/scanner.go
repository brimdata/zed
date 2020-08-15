package zngio

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync/atomic"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// zngScanner implements scanner.Scanner.
type zngScanner struct {
	ctx    context.Context
	reader *Reader
	filter filter.Filter
	finder *requiredPatternFinder
	span   nano.Span
	stats  scanner.ScannerStats
}

var _ scanner.ScannerAble = (*Reader)(nil)

func (r *Reader) NewScanner(ctx context.Context, filterExpr ast.BooleanExpr, s nano.Span) (scanner.Scanner, error) {
	var f filter.Filter
	var finder *requiredPatternFinder
	if filterExpr != nil {
		var err error
		if f, err = filter.Compile(filterExpr); err != nil {
			return nil, err
		}
		finder, err = newRequiredPatterFinder(filterExpr)
		if err != nil {
			return nil, err
		}
	}
	return &zngScanner{ctx: ctx, reader: r, filter: f, finder: finder, span: s}, nil
}

// Pull implements scanner.Scanner.Pull.
func (s *zngScanner) Pull() (zbuf.Batch, error) {
	for {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		id, buf, err := s.reader.readPayload()
		if buf == nil || err != nil {
			return nil, err
		}
		if id < 0 {
			if -id != zng.CtrlCompressed {
				// Discard everything else.
				continue
			}
			recs, err := s.scanUncompressed()
			if err != nil {
				return nil, err
			}
			if len(recs) == 0 {
				continue
			}
			return zbuf.NewArray(recs), nil
		}
		rec, err := s.scanOne(id, buf)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			return zbuf.NewArray([]*zng.Record{rec}), nil
		}
	}
}

func (s *zngScanner) scanUncompressed() ([]*zng.Record, error) {
	if s.finder != nil && !s.finder.find(s.reader.uncompressed.Bytes()) {
		// We know s.reader.uncompressed cannot contain any
		// records matching s.filter.
		s.reader.uncompressed = nil
		// xxx stats
		return nil, nil
	}
	var recs []*zng.Record
	for uncompressed := s.reader.uncompressed; uncompressed.Len() > 0; {
		id, err := readUvarint7(uncompressed)
		if err != nil {
			return nil, err
		}
		length, err := binary.ReadUvarint(uncompressed)
		if err != nil {
			return nil, err
		}
		raw := uncompressed.Next(int(length))
		if len(raw) < int(length) {
			return nil, errors.New("zngio: short read")
		}
		rec, err := s.scanOne(id, raw)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			recs = append(recs, rec)
		}
	}
	s.reader.uncompressed = nil
	return recs, nil
}

func (s *zngScanner) scanOne(id int, buf []byte) (*zng.Record, error) {
	rec, err := s.reader.parseValue(id, buf)
	if err != nil {
		return nil, err
	}
	atomic.AddInt64(&s.stats.BytesRead, int64(len(rec.Raw)))
	atomic.AddInt64(&s.stats.RecordsRead, 1)
	if s.span != nano.MaxSpan && !s.span.Contains(rec.Ts()) ||
		s.filter != nil && !s.filter(rec) {
		return nil, nil
	}
	atomic.AddInt64(&s.stats.BytesMatched, int64(len(rec.Raw)))
	atomic.AddInt64(&s.stats.RecordsMatched, 1)
	rec.CopyBody()
	return rec, nil
}

func readUvarint7(r io.ByteReader) (int, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if b >= 0x80 {
		return 0, errors.New("zngio: unexpected control message")
	}
	if (b & 0x40) == 0 {
		return int(b & 0x3f), nil
	}
	u64, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	return (int(u64) << 6) | int(b&0x3f), nil
}

// Stats implements scanner.Scanner.Stats.
func (s *zngScanner) Stats() *scanner.ScannerStats {
	return &scanner.ScannerStats{
		BytesRead:      atomic.LoadInt64(&s.stats.BytesRead),
		BytesMatched:   atomic.LoadInt64(&s.stats.BytesMatched),
		RecordsRead:    atomic.LoadInt64(&s.stats.RecordsRead),
		RecordsMatched: atomic.LoadInt64(&s.stats.RecordsMatched),
	}
}
