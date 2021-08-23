package csvio

import (
	"bytes"
	"io"

	"github.com/brimdata/zed/pkg/skim"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

// preProcess is a reader, meant to sit in front of the go csv reader, that
// looks for fields where quotes do not cover the entirety of a field. If such
// a case is found the quotes are stripped from the field.
//
// For instance, the line:
// field1,"field2" extra
// Would get converted into:
// field1,field2 extra
type preProcess struct {
	leftover []byte
	scanner  *skim.Scanner
}

func newPreProcess(r io.Reader) *preProcess {
	buffer := make([]byte, ReadSize)
	return &preProcess{
		scanner: skim.NewScanner(r, buffer, MaxLineSize),
	}
}

func (p *preProcess) Read(b []byte) (int, error) {
	n := len(p.leftover)
	if n > 0 {
		if cc := p.copy(b, p.leftover); cc < n {
			return cc, nil
		}
	}
	for {
		line, err := p.scanner.Scan()
		if len(line) > 0 {
			line = checkLine(line)
			cc := p.copy(b[n:], line)
			n += cc
			if cc < len(line) {
				return n, err
			}
		}
		if err != nil {
			return n, err
		}
	}
}

func (p *preProcess) copy(dst []byte, src []byte) int {
	cc := copy(dst, src)
	p.leftover = append(p.leftover[0:], src[cc:]...)
	return cc
}

func checkLine(line []byte) []byte {
	var field []byte
	var pos int
	for {
		comma := bytes.Index(line[pos:], []byte(","))
		newline := bytes.Index(line[pos:], []byte("\n"))
		if i := minIndex(comma, newline); i == -1 {
			field = line[pos:]
		} else {
			field = line[pos : pos+i+1]
		}
		old := len(field)
		if old == 0 {
			return line
		}
		field = checkField(field)
		n := len(field)
		pos += n
		if n != old {
			diff := old - n
			line = append(line[:pos], line[pos+diff:]...)
		}
	}
}

func minIndex(x, y int) int {
	switch {
	case x == -1 && y == -1:
		return -1
	case x == -1:
		return y
	case y == -1:
		return x
	case x < y:
		return x
	}
	return y
}

func checkField(field []byte) []byte {
	// Looking for the case where there's open and closed quotes and text to the
	// left or right side of it. If we find this case remove the quotes.
	for {
		start := bytes.IndexByte(field, '"')
		if start == -1 {
			return field
		}
		end := bytes.IndexByte(field[start+1:], '"')
		if end == -1 {
			return field
		}
		end += start + 1
		if start == 0 && end == len(field)-1 {
			return field
		}
		// also ignore double quotes
		if end == start+1 {
			return field
		}
		field = append(field[:end], field[end+1:]...)
		field = append(field[:start], field[start+1:]...)
	}
}
