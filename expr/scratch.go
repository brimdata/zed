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
	s.buf = zng.AppendInt(s.buf[:0], v)
	return s.buf
}

func (s *scratch) Uint(v uint64) zcode.Bytes {
	if s.keep {
		return zng.EncodeUint(v)
	}
	s.buf = zng.AppendUint(s.buf[0:], v)
	return s.buf
}

func (s *scratch) Float64(v float64) zcode.Bytes {
	if s.keep {
		return zng.EncodeFloat64(v)
	}
	s.buf = zng.AppendFloat64(s.buf[0:], v)
	return s.buf
}

func (s *scratch) Time(v nano.Ts) zcode.Bytes {
	if s.keep {
		return zng.EncodeTime(v)
	}
	s.buf = zng.AppendTime(s.buf[0:], v)
	return s.buf
}
