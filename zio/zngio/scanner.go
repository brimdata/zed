package zngio

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync/atomic"

	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// zngScanner implements scanner.Scanner.
type zngScanner struct {
	ctx          context.Context
	reader       *Reader
	bufferFilter *filter.BufferFilter
	filter       filter.Filter
	rec          zng.Record // Used to reduce memory allocations.
	span         nano.Span
	stats        scanner.ScannerStats
}

var _ scanner.ScannerAble = (*Reader)(nil)

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
			batch, err := s.scanUncompressed()
			if err != nil {
				return nil, err
			}
			if batch == nil {
				continue
			}
			return batch, nil
		}
		rec, err := s.scanOne(&s.rec, id, buf)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			rec.CopyBody()
			batch := newBatch(nil)
			batch.add(rec)
			return batch, nil
		}
	}
}

func (s *zngScanner) scanUncompressed() (zbuf.Batch, error) {
	ubuf := s.reader.uncompressedBuf
	s.reader.uncompressedBuf = nil
	if s.bufferFilter != nil && !s.bufferFilter.Eval(s.reader.zctx, ubuf.Bytes()) {
		// s.bufferFilter evaluated to false, so we know ubuf cannot
		// contain records matching s.filter.
		atomic.AddInt64(&s.stats.BytesRead, int64(ubuf.length()))
		ubuf.free()
		return nil, nil
	}
	batch := newBatch(ubuf)
	for ubuf.length() > 0 {
		id, err := readUvarint7(ubuf)
		if err != nil {
			return nil, err
		}
		length, err := binary.ReadUvarint(ubuf)
		if err != nil {
			return nil, err
		}
		raw := ubuf.next(int(length))
		if len(raw) < int(length) {
			return nil, errors.New("zngio: short read")
		}
		rec, err := s.scanOne(&s.rec, id, raw)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			batch.add(rec)
		}
	}
	if batch.Length() == 0 {
		batch.Unref()
		return nil, nil
	}
	return batch, nil
}

func (s *zngScanner) scanOne(rec *zng.Record, id int, buf []byte) (*zng.Record, error) {
	rec, err := s.reader.parseValue(rec, id, buf)
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
