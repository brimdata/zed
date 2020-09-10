package emitter

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zng"
)

type TypeLogger struct {
	io.WriteCloser
	verbose bool
}

func NewTypeLogger(path string, verbose bool) (*TypeLogger, error) {
	var f io.WriteCloser
	if path == "" {
		// Don't close stdout in case we live inside something
		// here that runs multiple instances of this to stdout.
		f = zio.NopCloser(os.Stdout)
	} else {
		var err error
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		file, err := fs.OpenFile(path, flags, 0600)
		if err != nil {
			return nil, err
		}
		f = file
	}
	return &TypeLogger{bufwriter.New(f), verbose}, nil
}

func (t *TypeLogger) Close() error {
	return t.WriteCloser.Close()

}
func (t *TypeLogger) TypeDef(id int, typ zng.Type) {
	var s string
	if t.verbose {
		s = formatType(typ) + "\n"
	} else {
		s = fmt.Sprintf("#%d:%s\n", id, typ)
	}
	t.Write([]byte(s))
}

func formatType(typ zng.Type) string {
	switch typ := typ.(type) {
	case *zng.TypeSet:
		return fmt.Sprintf("%d:set[<%d>]", typ.ID(), typ.InnerType.ID())
	case *zng.TypeArray:
		return fmt.Sprintf("%d:array[<%d>]", typ.ID(), typ.Type.ID())
	case *zng.TypeRecord:
		s := fmt.Sprintf("%d:record[", typ.ID())
		comma := ""
		for _, col := range typ.Columns {
			s += fmt.Sprintf("%s%s:<%d>", comma, col.Name, col.Type.ID())
			comma = ","
		}
		s += "]"
		return s
	default:
		// this shouldn't happen
		return strconv.Itoa(typ.ID())
	}
}
