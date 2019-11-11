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
	columns      []zeek.Column
}

type parser struct {
	header
	unknown    int
	needfields bool
	needtypes  bool
	descriptor *zson.Descriptor
	addPath    bool
}

var (
	ErrBadRecordDef = errors.New("bad types/fields definition in zeek header")
	ErrBadFormat    = errors.New("bad format")          //XXX
	ErrBadValue     = errors.New("bad value")           //XXX
	ErrBadEscape    = errors.New("bad escape sequence") //XXX
)

func newParser() *parser {
	return &parser{header: header{separator: " "}}
}

func badfield(field string) error {
	return fmt.Errorf("encountered bad header field %s parsing zeek log", field)
}

func (c *parser) parseFields(fields []string) error {
	if len(c.columns) != len(fields) {
		c.columns = make([]zeek.Column, len(fields))
		c.needtypes = true
	}
	for k, field := range fields {
		//XXX check that string conforms to a field name syntax
		c.columns[k].Name = field
	}
	c.needfields = false
	c.descriptor = nil
	return nil
}

func (c *parser) parseTypes(types []string) error {
	if len(c.columns) != len(types) {
		c.columns = make([]zeek.Column, len(types))
		c.needfields = true
	}
	for k, name := range types {
		typ, err := zeek.LookupType(name)
		if err != nil {
			return err
		}
		c.columns[k].Type = typ
	}
	c.needtypes = false
	c.descriptor = nil
	return nil
}

func (c *parser) parseDirective(line []byte) error {
	if len(line) == 0 {
		return ErrBadFormat
	}
	// skip '#'
	line = line[1:]
	if len(line) == 0 {
		return ErrBadFormat
	}
	tokens := strings.Split(string(line), c.separator)
	switch tokens[0] {
	case "separator":
		if len(tokens) != 2 {
			return badfield("separator")
		}
		c.separator = string(zeek.Unescape([]byte(tokens[1])))
	case "set_separator":
		if len(tokens) != 2 {
			return badfield("set_separator")
		}
		c.setSeparator = tokens[1]
	case "empty_field":
		if len(tokens) != 2 {
			return badfield("empty_field")
		}
		//XXX this should be ok now as we process on ingest
		if tokens[1] != "(empty)" {
			return badfield(fmt.Sprintf("#empty_field (non-standard value '%s')", tokens[1]))
		}
		c.emptyField = tokens[1]
	case "unset_field":
		if len(tokens) != 2 {
			return badfield("unset_field")
		}
		//XXX this should be ok now as we process on ingest
		if tokens[1] != "-" {
			return badfield(fmt.Sprintf("#unset_field (non-standard value '%s')", tokens[1]))
		}
		c.unsetField = tokens[1]
	case "path":
		if len(tokens) != 2 {
			return badfield("path")
		}
		c.path = tokens[1]
	case "open":
		if len(tokens) != 2 {
			return badfield("open")
		}
		c.open = tokens[1]
	case "close":
		if len(tokens) != 2 {
			return badfield("close")
		}
		c.open = tokens[1]
	case "fields":
		if len(tokens) < 2 {
			return badfield("fields")
		}
		if err := c.parseFields(tokens[1:]); err != nil {
			return err
		}
	case "types":
		if len(tokens) < 2 {
			return badfield("types")
		}
		if err := c.parseTypes(tokens[1:]); err != nil {
			return err
		}
	default:
		c.unknown++
	}
	return nil
}

func (c *parser) lookup(r *resolver.Table) (*zson.Descriptor, error) {
	// add descriptor and _path, form the columns, and lookup the td
	// in the space's descriptor table.  If it is a new descriptor, we
	// persist the space's state to disk
	if len(c.columns) == 0 || c.needfields || c.needtypes {
		return nil, ErrBadRecordDef
	}
	cols := c.columns
	if !c.hasField("_path", nil) {
		pathcol := zeek.Column{Name: "_path", Type: zeek.TypeString}
		cols = append([]zeek.Column{pathcol}, cols...)
		c.addPath = true
	} else {
		c.addPath = false
	}
	return r.GetByColumns(cols), nil
}

func (c *parser) hasField(name string, typ zeek.Type) bool {
	for _, c := range c.columns {
		if name == c.Name && (typ == nil || c.Type == typ) {
			return true
		}
	}
	return false
}

func (c *parser) hasTs() bool {
	return c.hasField("ts", zeek.TypeTime)
}

func (p *parser) parseValue(line []byte, r *resolver.Table) (*zson.Record, error) {
	if p.descriptor == nil {
		d, err := p.lookup(r)
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
