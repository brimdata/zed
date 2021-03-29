package zjsonio

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/brimdata/zq/pkg/joe"
	"github.com/brimdata/zq/pkg/skim"
	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

type Reader struct {
	scanner *skim.Scanner
	zctx    *resolver.Context
	mapper  map[int]*zng.TypeRecord
	builder *zcode.Builder
}

func NewReader(reader io.Reader, zctx *resolver.Context) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner: skim.NewScanner(reader, buffer, MaxLineSize),
		zctx:    zctx,
		mapper:  make(map[int]*zng.TypeRecord),
		builder: zcode.NewBuilder(),
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	e := func(err error) error {
		if err == nil {
			return err
		}
		return fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}

	line, err := r.scanner.ScanLine()
	if line == nil {
		return nil, e(err)
	}
	var v Record
	if err := json.Unmarshal(line, &v); err != nil {
		return nil, e(err)
	}
	var recType *zng.TypeRecord
	if v.Type == nil {
		var ok bool
		recType, ok = r.mapper[v.Id]
		if !ok {
			return nil, fmt.Errorf("undefined type ID: %d", v.Id)
		}
	} else {
		if v.Aliases != nil {
			if err := r.parseAliases(v.Aliases); err != nil {
				return nil, err
			}
		}
		recType, err = decodeTypeRecord(r.zctx, v.Type)
		if err != nil {
			return nil, err
		}
		r.mapper[v.Id] = recType
	}
	r.builder.Reset()
	if err := decodeRecord(r.builder, recType, v.Values); err != nil {
		return nil, e(err)
	}
	zv, err := r.builder.Bytes().ContainerBody()
	if err != nil {
		return nil, e(err)
	}
	return zng.NewRecordCheck(recType, zv)
}

func (r *Reader) parseAliases(aliases []Alias) error {
	for _, alias := range aliases {
		typ, err := decodeTypeAny(r.zctx, alias.Type.(joe.Interface))
		if err != nil {
			return fmt.Errorf("error decoding alias type: \"%s\"", err)
		}
		_, err = r.zctx.LookupTypeAlias(alias.Name, typ)
		if err != nil {
			return err
		}
	}
	return nil
}
