package zeek

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/mccanne/zq/pkg/nano"
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
	resolver    *resolver.Table
	descriptors map[int]*zson.Descriptor
	unknown     int  // Count of unknown directives

	// Members below (plus header above) are all state related to
	// parsing legacy zeek.  legacyVal indicates whether the last
	// parsed directive was legacy or not.  As described in the
	// zson spec, this governs whether values should be parsed as
	// legacy or not.
	// legacyDesc is a lazily-allocated Descriptor corresponding
	// to the contents of the #fields and #types directives.
	needfields  bool
	needtypes   bool
	legacyDesc *zson.Descriptor
	legacyVal   bool
}

var (
	ErrBadRecordDef     = errors.New("bad types/fields definition in zeek header")
	ErrBadFormat        = errors.New("bad format")          //XXX
	ErrBadValue         = errors.New("bad value")           //XXX
	ErrBadEscape        = errors.New("bad escape sequence") //XXX
	ErrDescriptorExists = errors.New("descriptor already exists")
	ErrInvalidDesc      = errors.New("invalid descriptor")
)

func newParser(r *resolver.Table) *parser {
	return &parser{
		header:      header{separator: " "},
		resolver:    r,
		descriptors: make(map[int]*zson.Descriptor),
	}
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
	c.legacyDesc = nil
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
	c.legacyDesc = nil
	return nil
}

func parseLeadingInt(line []byte) (val int, rest []byte, err error) {
	i := bytes.IndexByte(line, byte(':'))
	if i < 0 {
		return -1, nil, ErrBadFormat
	}
	v, err := strconv.ParseUint(string(line[:i]), 10, 32)
	if err != nil {
		return -1, nil, err
	}
	return int(v), line[i+1:], nil
}

func (p *parser) parseDescriptor(line []byte) error {
	// #int:type
	descriptor, rest, err := parseLeadingInt(line)
	if err != nil {
		return err
	}

	_, exists := p.descriptors[descriptor]
	if exists {
		return ErrDescriptorExists
	}

	// XXX doesn't handle nested descriptors such as
	// #1:record[foo:int]
	// #2:record[foos:vector[1]]
	typ, err := zeek.LookupType(string(rest))
	if err != nil {
		return err
	}

	recordType, ok := typ.(*zeek.TypeRecord)
	if !ok {
		return ErrBadValue // XXX?
	}

	p.descriptors[descriptor] = p.resolver.GetByValue(recordType)
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

	if line[0] == '!' {
		// comment
		c.legacyVal = false
		return nil
	}

	if line[0] >= '1' && line[0] <= '9' {
		c.legacyVal = false
		return c.parseDescriptor(line)
	}

	tokens := strings.Split(string(line), c.separator)
	switch tokens[0] {
	case "separator":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("separator")
		}
		c.separator = string(zeek.Unescape([]byte(tokens[1])))
	case "set_separator":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("set_separator")
		}
		c.setSeparator = tokens[1]
	case "empty_field":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("empty_field")
		}
		//XXX this should be ok now as we process on ingest
		if tokens[1] != "(empty)" {
			return badfield(fmt.Sprintf("#empty_field (non-standard value '%s')", tokens[1]))
		}
		c.emptyField = tokens[1]
	case "unset_field":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("unset_field")
		}
		//XXX this should be ok now as we process on ingest
		if tokens[1] != "-" {
			return badfield(fmt.Sprintf("#unset_field (non-standard value '%s')", tokens[1]))
		}
		c.unsetField = tokens[1]
	case "path":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("path")
		}
		c.path = tokens[1]
	case "open":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("open")
		}
		c.open = tokens[1]
	case "close":
		c.legacyVal = true
		if len(tokens) != 2 {
			return badfield("close")
		}
		c.open = tokens[1]
	case "fields":
		c.legacyVal = true
		if len(tokens) < 2 {
			return badfield("fields")
		}
		if err := c.parseFields(tokens[1:]); err != nil {
			return err
		}
	case "types":
		c.legacyVal = true
		if len(tokens) < 2 {
			return badfield("types")
		}
		if err := c.parseTypes(tokens[1:]); err != nil {
			return err
		}
	case "sort":
		// #sort [+-]<field>,[+-]<field>,...
		// XXX handle me
		c.legacyVal = false
	default:
		c.unknown++
	}
	return nil
}

func (c *parser) lookup() (*zson.Descriptor, error) {
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
	}
	return c.resolver.GetByColumns(cols), nil
}

func (p *parser) findField(name string, typ zeek.Type) int {
	for i, c := range p.columns {
		if name == c.Name && (typ == nil || c.Type == typ) {
			return i
		}
	}
	return -1
}

func (c *parser) hasField(name string, typ zeek.Type) bool {
	return c.findField(name, typ) >= 0
}

func (p *parser) parseLegacyValue(line []byte) (*zson.Record, error) {
	if p.legacyDesc == nil {
		d, err := p.lookup()
		if err != nil {
			return nil, err
		}
		p.legacyDesc = d
	}
	tsCol, ok := p.legacyDesc.ColumnOfField("ts")
	if !ok {
		tsCol = -1
	}
	raw, ts, err := zson.NewRawAndTsFromZeekTSV(p.legacyDesc, tsCol, []byte(p.path), line)
	if err != nil {
		return nil, err
	}
	return zson.NewRecord(p.legacyDesc, ts, raw), nil
}

func (p *parser) parseValue(line []byte) (*zson.Record, error) {
	if p.legacyVal {
		return p.parseLegacyValue(line)
	}

	// From the zson spec:
	// A regular value is encoded on a line as type descriptor
	// followed by ":" followed by a value encoding.
	id, rest, err := parseLeadingInt(line)
	if err != nil {
		return nil, err
	}

	descriptor, ok := p.descriptors[id]
	if !ok {
		return nil, ErrInvalidDesc
	}

	raw, err := zson.NewRawFromZSON(descriptor, rest)
	if err != nil {
		return nil, err
	}

	record, err := zson.NewRecord(descriptor, nano.MinTs, raw), nil
	if err != nil {
		return nil, err
	}
	ts, err := record.AccessTime("ts")
	if err == nil {
		record.Ts = ts
	}
	// Ignore errors, it just means the point doesn't have a ts field
	return record, nil
}
