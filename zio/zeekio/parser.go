package zeekio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
)

type header struct {
	separator    string
	setSeparator string
	emptyField   string
	unsetField   string
	Path         string
	open         string
	close        string
	columns      []zed.Column
}

type Parser struct {
	header
	zctx       *zed.Context
	unknown    int // Count of unknown directives
	needfields bool
	needtypes  bool
	addpath    bool
	// descriptor is a lazily-allocated Descriptor corresponding
	// to the contents of the #fields and #types directives.
	descriptor   *zed.TypeRecord
	builder      builder
	sourceFields []int
}

var ErrBadRecordDef = errors.New("bad types/fields definition in zeek header")

func NewParser(r *zed.Context) *Parser {
	return &Parser{
		header: header{separator: " "},
		zctx:   r,
	}
}

func badfield(field string) error {
	return fmt.Errorf("encountered bad header field %s parsing zeek log", field)
}

func (p *Parser) parseFields(fields []string) error {
	if len(p.columns) != len(fields) {
		p.columns = make([]zed.Column, len(fields))
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
		p.columns = make([]zed.Column, len(types))
		p.needfields = true
	}
	for k, name := range types {
		typ, err := p.parseType(name)
		if err != nil {
			return err
		}
		if !isValidInputType(typ) {
			return ErrIncompatibleZeekType
		}
		p.columns[k].Type = typ
	}
	p.needtypes = false
	p.descriptor = nil
	return nil
}

func isValidInputType(typ zed.Type) bool {
	switch t := typ.(type) {
	case *zed.TypeRecord, *zed.TypeUnion:
		return false
	case *zed.TypeSet:
		return isValidInputType(t.Type)
	case *zed.TypeArray:
		return isValidInputType(t.Type)
	default:
		return true
	}
}

func (p *Parser) parseType(in string) (zed.Type, error) {
	in = strings.TrimSpace(in)
	if words := strings.SplitN(in, "[", 2); len(words) == 2 && strings.HasSuffix(words[1], "]") {
		if typ, err := p.parsePrimitiveType(strings.TrimSuffix(words[1], "]")); err == nil {
			if words[0] == "set" {
				return p.zctx.LookupTypeSet(typ), nil
			}
			if words[0] == "vector" {
				return p.zctx.LookupTypeArray(typ), nil
			}
		}
	}
	return p.parsePrimitiveType(in)
}

func (p *Parser) parsePrimitiveType(in string) (zed.Type, error) {
	in = strings.TrimSpace(in)
	switch in {
	case "addr":
		return zed.TypeIP, nil
	case "bool":
		return zed.TypeBool, nil
	case "count":
		return zed.TypeUint64, nil
	case "double":
		return zed.TypeFloat64, nil
	case "enum":
		return p.zctx.LookupTypeAlias("zenum", zed.TypeString)
	case "int":
		return zed.TypeInt64, nil
	case "interval":
		return zed.TypeDuration, nil
	case "port":
		return p.zctx.LookupTypeAlias("port", zed.TypeUint16)
	case "string":
		return zed.TypeString, nil
	case "subnet":
		return zed.TypeNet, nil
	case "time":
		return zed.TypeTime, nil
	}
	return nil, fmt.Errorf("unknown type: %s", in)
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
		p.separator = string(unescapeZeekString([]byte(tokens[1])))
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
		p.Path = tokens[1]
		if p.Path == "-" {
			p.Path = ""
		}
	case "open":
		if len(tokens) != 2 {
			return badfield("open")
		}
		p.open = tokens[1]
	case "close":
		if len(tokens) > 2 {
			return badfield("close")
		}
		if len(tokens) == 1 {
			p.close = ""
		} else {
			p.close = tokens[1]
		}

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

// Unflatten() turns a set of columns from legacy zeek logs into a
// zng-compatible format by creating nested records for any dotted
// field names. If addpath is true, a _path column is added if not
// already present. The columns are returned as a slice along with a
// bool indicating if a _path column was added.
// Note that according to the zng spec, all the fields for a nested
// record must be adjacent which simplifies the logic here.
func Unflatten(zctx *zed.Context, columns []zed.Column, addPath bool) ([]zed.Column, bool, error) {
	hasPath := false
	for _, col := range columns {
		// XXX could validate field names here...
		if col.Name == "_path" {
			hasPath = true
		}
	}
	out, err := unflattenRecord(zctx, columns)
	if err != nil {
		return nil, false, err
	}

	var needpath bool
	if addPath && !hasPath {
		pathcol := zed.NewColumn("_path", zed.TypeString)
		out = append([]zed.Column{pathcol}, out...)
		needpath = true
	}
	return out, needpath, nil
}

func unflattenRecord(zctx *zed.Context, cols []zed.Column) ([]zed.Column, error) {
	// extract a []Column consisting of all the leading columns
	// from the input that belong to the same record, with the
	// common prefix removed from their name.
	// returns the prefix and the extracted same-record columns.
	recCols := func(cols []zed.Column) (string, []zed.Column) {
		var ret []zed.Column
		var prefix string
		if dot := strings.IndexByte(cols[0].Name, '.'); dot != -1 {
			prefix = cols[0].Name[:dot]
		}
		for i := range cols {
			if !strings.HasPrefix(cols[i].Name, prefix+".") {
				break
			}
			trimmed := strings.TrimPrefix(cols[i].Name, prefix+".")
			ret = append(ret, zed.NewColumn(trimmed, cols[i].Type))
		}
		return prefix, ret
	}
	var out []zed.Column
	i := 0
	for i < len(cols) {
		col := cols[i]
		if strings.IndexByte(col.Name, '.') < 0 {
			// Just a top-level field.
			out = append(out, col)
			i++
			continue
		}
		prefix, nestedCols := recCols(cols[i:])
		recCols, err := unflattenRecord(zctx, nestedCols)
		if err != nil {
			return nil, err
		}
		recType, err := zctx.LookupTypeRecord(recCols)
		if err != nil {
			return nil, err
		}
		out = append(out, zed.NewColumn(prefix, recType))
		i += len(nestedCols)
	}
	return out, nil
}

func (p *Parser) setDescriptor() error {
	// add descriptor and _path, form the columns, and lookup the td
	// in the space's descriptor table.
	if len(p.columns) == 0 || p.needfields || p.needtypes {
		return ErrBadRecordDef
	}
	cols, sourceFields := coalesceRecordColumns(p.columns)
	cols, addpath, err := Unflatten(p.zctx, cols, p.Path != "")
	if err != nil {
		return err
	}
	p.descriptor, err = p.zctx.LookupTypeRecord(cols)
	if err != nil {
		return err
	}
	p.addpath = addpath
	p.sourceFields = sourceFields
	return nil
}

// coalesceRecordColumns returns a permutation of cols in which the columns of
// each nested record have been made adjacent along with a slice containing the
// index of the source field for each column in that permutation.
func coalesceRecordColumns(cols []zed.Column) ([]zed.Column, []int) {
	prefixes := map[string]bool{"": true}
	var outcols []zed.Column
	var sourceFields []int
	for i, c := range cols {
		outcols = append(outcols, c)
		sourceFields = append(sourceFields, i)
		prefix := getPrefix(c.Name)
		for !prefixes[prefix] {
			prefixes[prefix] = true
			prefix = getPrefix(prefix)
		}
		if prefix != "" {
			for j := i; j > 0; j-- {
				if strings.HasPrefix(outcols[j-1].Name, prefix) {
					// Insert c at j.
					copy(outcols[j+1:], outcols[j:])
					outcols[j] = c
					copy(sourceFields[j+1:], sourceFields[j:])
					sourceFields[j] = i
					break
				}
			}
		}
	}
	return outcols, sourceFields
}

// getPrefix returns the prefix of dotpath up to and including its final period.
// If dotpath does not contain a period, getPrefix returns the empty string.
func getPrefix(name string) string {
	name = strings.TrimRight(name, ".")
	i := strings.LastIndex(name, ".")
	if i < 0 {
		return ""
	}
	return name[:i+1]
}

// Descriptor returns the current descriptor (from the most recently
// seen #types and #fields lines) and a bool indicating whether _path
// was added to the descriptor. If no descriptor is present, nil and
// and false are returned.
func (p *Parser) Descriptor() (*zed.TypeRecord, bool) {
	if p.descriptor != nil {
		return p.descriptor, p.addpath
	}
	if err := p.setDescriptor(); err != nil {
		return nil, false
	}
	return p.descriptor, p.addpath

}

func (p *Parser) ParseValue(line []byte) (*zed.Value, error) {
	if p.descriptor == nil {
		if err := p.setDescriptor(); err != nil {
			return nil, err
		}
	}
	var path []byte
	if p.Path != "" && p.addpath {
		//XXX should store path as a byte slice so it doens't get copied
		// each time here
		path = []byte(p.Path)
	}
	return p.builder.build(p.descriptor, p.sourceFields, path, line)
}
