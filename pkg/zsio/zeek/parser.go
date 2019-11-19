package zeek

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
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
	descriptor *zson.Descriptor
}

var (
	ErrBadRecordDef = errors.New("bad types/fields definition in zeek header")
	ErrBadEscape    = errors.New("bad escape sequence") //XXX
)

func NewParser(r *resolver.Table) *Parser {
	return &Parser{
		header:   header{separator: " "},
		resolver: r,
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

func (p *Parser) lookup() (*zson.Descriptor, error) {
	// add descriptor and _path, form the columns, and lookup the td
	// in the space's descriptor table.
	if len(p.columns) == 0 || p.needfields || p.needtypes {
		return nil, ErrBadRecordDef
	}
	cols := p.columns
	if !p.hasField("_path", nil) {
		pathcol := zeek.Column{Name: "_path", Type: zeek.TypeString}
		cols = append([]zeek.Column{pathcol}, cols...)
	}
	return p.resolver.GetByColumns(cols), nil
}

func (p *Parser) findField(name string, typ zeek.Type) int {
	for i, c := range p.columns {
		if name == c.Name && (typ == nil || c.Type == typ) {
			return i
		}
	}
	return -1
}

func (p *Parser) hasField(name string, typ zeek.Type) bool {
	return p.findField(name, typ) >= 0
}

func (p *Parser) ParseValue(line []byte) (*zson.Record, error) {
	if p.descriptor == nil {
		d, err := p.lookup()
		if err != nil {
			return nil, err
		}
		p.descriptor = d
	}
	tsCol, ok := p.descriptor.ColumnOfField("ts")
	if !ok {
		tsCol = -1
	}
	raw, ts, err := zson.NewRawAndTsFromZeekTSV(p.descriptor, tsCol, []byte(p.path), line)
	if err != nil {
		return nil, err
	}
	return zson.NewRecord(p.descriptor, ts, raw), nil
}
