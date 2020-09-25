package expr

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type scratch struct {
	keep bool
	buf  zcode.Bytes
}

func (s *scratch) Int(v int64) zcode.Bytes {
	if s.keep {
		return zng.EncodeInt(v)
	}
	b := s.buf
	if b != nil {
		b = b[:0]
	}
	s.buf = zng.AppendInt(b, v)
	if len(s.buf) == 0 {
		return nil
	}
	return s.buf
}

func (s *scratch) Uint(v uint64) zcode.Bytes {
	if s.keep {
		return zng.EncodeUint(v)
	}
	b := s.buf
	if b != nil {
		b = b[:0]
	}
	s.buf = zng.AppendUint(b, v)
	if len(s.buf) == 0 {
		return nil
	}
	return s.buf
}

func (s *scratch) Float64(v float64) zcode.Bytes {
	if s.keep {
		return zng.EncodeFloat64(v)
	}
	b := s.buf
	if b != nil {
		b = b[:0]
	}
	s.buf = zng.AppendFloat64(b, v)
	return s.buf
}

func (s *scratch) Time(v nano.Ts) zcode.Bytes {
	if s.keep {
		return zng.EncodeTime(v)
	}
	b := s.buf
	if b != nil {
		b = b[:0]
	}
	s.buf = zng.AppendTime(b, v)
	if len(s.buf) == 0 {
		return nil
	}
	return s.buf
}
