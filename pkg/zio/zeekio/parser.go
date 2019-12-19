package zeekio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/pkg/zng/resolver"
	"github.com/mccanne/zq/pkg/zval"
)

type header struct {
	separator    string
	setSeparator string
	emptyField   string
	unsetField   string
	path         string
	open         string
	close        string
	columns      []zeek.Column
}

type Parser struct {
	header
	resolver   *resolver.Table
	unknown    int // Count of unknown directives
	needfields bool
	needtypes  bool
	// descriptor is a lazily-allocated Descriptor corresponding
	// to the contents of the #fields and #types directives.
	descriptor *zng.Descriptor
	builder    *zval.Builder
}

var (
	ErrBadRecordDef = errors.New("bad types/fields definition in zeek header")
	ErrBadEscape    = errors.New("bad escape sequence") //XXX
)

func NewParser(r *resolver.Table) *Parser {
	return &Parser{
		header:   header{separator: " "},
		resolver: r,
		builder:  zval.NewBuilder(),
	}
}

func badfield(field string) error {
	return fmt.Errorf("encountered bad header field %s parsing zeek log", field)
}

func (p *Parser) parseFields(fields []string) error {
	if len(p.columns) != len(fields) {
		p.columns = make([]zeek.Column, len(fields))
		p.needtypes = true
	}
	for k, field := range fields {
		//XXX check that string conforms to a field name syntax
		p.columns[k].Name = field
	}
	p.needfields = false
	p.descriptor = nil
	return nil
}

func (p *Parser) parseTypes(types []string) error {
	if len(p.columns) != len(types) {
		p.columns = make([]zeek.Column, len(types))
		p.needfields = true
	}
	for k, name := range types {
		typ, err := zeek.LookupType(name)
		if err != nil {
			return err
		}
		p.columns[k].Type = typ
	}
	p.needtypes = false
	p.descriptor = nil
	return nil
}

func (p *Parser) ParseDirective(line []byte) error {
	if line[0] == '#' {
		line = line[1:]
	}
	tokens := strings.Split(string(line), p.separator)
	switch tokens[0] {
	case "separator":
		if len(tokens) != 2 {
			return badfield("separator")
		}
		p.separator = string(zeek.Unescape([]byte(tokens[1])))
	case "set_separator":
		if len(tokens) != 2 {
			return badfield("set_separator")
		}
		p.setSeparator = tokens[1]
	case "empty_field":
		if len(tokens) != 2 {
			return badfield("empty_field")
		}
		//XXX this should be ok now as we process on ingest
		if tokens[1] != "(empty)" {
			return badfield(fmt.Sprintf("#empty_field (non-standard value '%s')", tokens[1]))
		}
		p.emptyField = tokens[1]
	case "unset_field":
		if len(tokens) != 2 {
			return badfield("unset_field")
		}
		//XXX this should be ok now as we process on ingest
		if tokens[1] != "-" {
			return badfield(fmt.Sprintf("#unset_field (non-standard value '%s')", tokens[1]))
		}
		p.unsetField = tokens[1]
	case "path":
		if len(tokens) != 2 {
			return badfield("path")
		}
		p.path = tokens[1]
		if p.path == "-" {
			p.path = ""
		}
	case "open":
		if len(tokens) != 2 {
			return badfield("open")
		}
		p.open = tokens[1]
	case "close":
		if len(tokens) != 2 {
			return badfield("close")
		}
		p.close = tokens[1]
	case "fields":
		if len(tokens) < 2 {
			return badfield("fields")
		}
		if err := p.parseFields(tokens[1:]); err != nil {
			return err
		}
	case "types":
		if len(tokens) < 2 {
			return badfield("types")
		}
		if err := p.parseTypes(tokens[1:]); err != nil {
			return err
		}
	default:
		// XXX return an error?
		p.unknown++
	}
	return nil
}

// Unflatten() turns a set of columns from legacy zeek logs into
// a zng-compatible format by creating nested records for any
// dotted field names and adding a _path column if one is not already
// present.  Note that according to the zng spec, all the fields for
// a nested record must be adjacent which simplifies the logic here.
func Unflatten(columns []zeek.Column, addPath bool) ([]zeek.Column, error) {
	hasPath := false
	cols := make([]zeek.Column, 0)
	var nestedCols []zeek.Column
	var nestedField string
	for _, col := range columns {
		// XXX could validate field names here...
		if col.Name == "_path" {
			hasPath = true
		}

		var fld string
		dot := strings.IndexByte(col.Name, '.')
		if dot >= 0 {
			fld = col.Name[:dot]
		}

		// Check if we're entering or leaving a nested record.
		if fld != nestedField {
			if len(nestedField) > 0 {
				// We've reached the end of a nested record.
				recType := zeek.LookupTypeRecord(nestedCols)
				newcol := zeek.Column{nestedField, recType}
				cols = append(cols, newcol)
			}

			if len(fld) > 0 {
				// We're entering a new nested record.
				nestedCols = make([]zeek.Column, 0)
			}
			nestedField = fld
		}

		if len(fld) == 0 {
			// Just a regular field.
			cols = append(cols, col)
		} else {
			// Add to the nested record.
			newcol := zeek.Column{col.Name[dot+1:], col.Type}
			nestedCols = append(nestedCols, newcol)
		}
	}

	// If we were in the midst of a nested record, make sure we
	// account for it.
	if len(nestedField) > 0 {
		recType := zeek.LookupTypeRecord(nestedCols)
		newcol := zeek.Column{nestedField, recType}
		cols = append(cols, newcol)
	}

	if addPath && !hasPath {
		pathcol := zeek.Column{Name: "_path", Type: zeek.TypeString}
		cols = append([]zeek.Column{pathcol}, cols...)
	}
	return cols, nil
}

func (p *Parser) lookup() (*zng.Descriptor, error) {
	// add descriptor and _path, form the columns, and lookup the td
	// in the space's descriptor table.
	if len(p.columns) == 0 || p.needfields || p.needtypes {
		return nil, ErrBadRecordDef
	}

	cols, err := Unflatten(p.columns, p.path != "")
	if err != nil {
		return nil, err
	}
	return p.resolver.GetByColumns(cols), nil
}

func (p *Parser) ParseValue(line []byte) (*zng.Record, error) {
	if p.descriptor == nil {
		d, err := p.lookup()
		if err != nil {
			return nil, err
		}
		p.descriptor = d
	}
	var path []byte
	if p.path != "" {
		//XXX should store path as a byte slice so it doens't get copied
		// each time here
		path = []byte(p.path)
	}
	zv, ts, err := zng.NewRawAndTsFromZeekTSV(p.builder, p.descriptor, path, line)
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(p.descriptor, ts, zv), nil
}
