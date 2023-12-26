package dynfield

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

type Path []zed.Value

func (p Path) Append(b []byte) []byte {
	for i, v := range p {
		if i > 0 {
			b = append(b, 0)
		}
		b = append(b, v.Bytes()...)
	}
	return b
}

func (p Path) String() string {
	var b []byte
	for i, v := range p {
		if i > 0 {
			b = append(b, '.')
		}
		b = append(b, zson.FormatValue(&v)...)
	}
	return string(b)
}

type List []Path

func (l List) Append(b []byte) []byte {
	for i, path := range l {
		if i > 0 {
			b = append(b, ',')
		}
		b = path.Append(b)
	}
	return b
}
