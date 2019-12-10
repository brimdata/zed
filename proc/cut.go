package proc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zval"
)

var ErrNoField = errors.New("cut field not found")

// fieldInfo encodes the structure of a particular "cut" invocation in a
// format that enables the runtime processing to happen as efficiently
// as possible.  When handling an input record, we build an output record
// using a zval.Builder but when handling fields within nested records,
// calls to BeginContainer() and EndContainer() on the builder need to
// happen at the right times to yield the proper output structure.
// This is probably best illustrated with an example, consider the proc
// "cut a, b.c, b.d, x.y.z".
//
// At runtime, this needs to turn into the following actions:
// 1.  builder.Append([value of a from the input record])
// 2.  builder.BeginContainer()  // for "b"
// 3.  builder.Append([value of b.c from the input record])
// 4.  builder.Append([value of b.d from the input record])
// 5.  builder.EndContainer()    // for "b"
// 6.  builder.BeginContainer()  // for "x"
// 7.  builder.BeginContainer()  // for "x.y"
// 8.  builder.Append([value of x.y.z. from the input record])
// 9.  builder.EndContainer()    // for "x.y"
// 10. builder.EndContainer()    // for "y"
//
// This is encoded into the following fieldInfo objects:
//  {name: "a", containerBegins: [], containerEnds: 0}         // step 1
//  {name: "c", containerBegins: ["b"], containerEnds: 0}      // steps 2-3
//  {name: "d", containerBegins: [], containerEnds: 1     }    // steps 4-5
//  {name: "z", containerBegins: ["x", "y"], containerEnds: 2} // steps 6-10
type fieldInfo struct {
	resolver        expr.FieldExprResolver
	name            string
	fullname        string
	containerBegins []string
	containerEnds   int
}

type Cut struct {
	Base
	fields   []fieldInfo
	cutmap   map[int]*zson.Descriptor
	nblocked int
	builder  *zval.Builder
}

// Build the structures we need to construct output records efficiently.
// See the comment above for a description of the desired output.
// Note that we require any nested fields from the same parent record
// to be adjacent.  Alternatively we could re-order provided fields
// so the output record can be constructed efficiently, though we don't
// do this now since it might confuse users who expect to see output
// fields in the order they specified.
func CompileCutProc(c *Context, parent Proc, node *ast.CutProc) (*Cut, error) {
	seenRecords := make(map[string]bool)
	fieldInfos := make([]fieldInfo, 0, len(node.Fields))
	var currentRecord []string
	for i, field := range node.Fields {
		resolver, err := expr.CompileFieldExpr(field)
		if err != nil {
			return nil, err
		}

		names, err := split(field)
		if err != nil {
			return nil, err
		}

		// Grab everything except the leaf field name and see if
		// it has changed from the previous field.  If it hasn't,
		// things are simple but if it has, we need to carefully
		// figure out which records we are stepping in and out of.
		record := names[:len(names)-1]
		var containerBegins []string
		if !sameRecord(record, currentRecord) {
			// currentRecord is what nested record the zval.Builder
			// is currently working on, record is the nested
			// record for the current field.  First figure out
			// what (if any) common parents are shared.
			l := len(currentRecord)
			if len(record) < l {
				l = len(record)
			}
			pos := 0
			for pos < l {
				if record[pos] != currentRecord[pos] {
					break
				}
				pos += 1
			}

			// Note any previously encoded records that are
			// now finished.
			if i > 0 {
				fieldInfos[i-1].containerEnds = len(currentRecord) - pos
			}

			// Validate any new records that we're starting
			// (i.e., ensure that we didn't handle fields from
			// the same record previously), then record the names
			// of all these records.
			for pos2 := pos; pos2 < len(record); pos2++ {
				recname := strings.Join(record[:pos2+1], ".")
				_, seen := seenRecords[recname]
				if seen {
					return nil, fmt.Errorf("All cut fields in record %s must be adjacent", recname)
				}
				seenRecords[recname] = true
				containerBegins = append(containerBegins, record[pos2])
			}
			currentRecord = record
		}
		fullname := strings.Join(names, ".")
		fname := names[len(names)-1]
		fieldInfos = append(fieldInfos, fieldInfo{resolver, fname, fullname, containerBegins, 0})
	}
	fieldInfos[len(fieldInfos)-1].containerEnds = len(currentRecord)

	return &Cut{
		Base:    Base{Context: c, Parent: parent},
		fields:  fieldInfos,
		cutmap:  make(map[int]*zson.Descriptor),
		builder: zval.NewBuilder(),
	}, nil
}

// Split an ast.FieldExpr representing a chain of record field references
// into a list of strings representing the names.
// E.g., "x.y.z" -> ["x", "y", "z"]
func split(node ast.FieldExpr) ([]string, error) {
	switch n := node.(type) {
	case *ast.FieldRead:
		return []string{n.Field}, nil
	case *ast.FieldCall:
		if n.Fn != "RecordFieldRead" {
			return nil, fmt.Errorf("unexpected field op %s", n.Fn)
		}
		names, err := split(n.Field)
		if err != nil {
			return nil, err
		}
		return append(names, n.Param), nil
	default:
		return nil, fmt.Errorf("unexpected node type %T", node)
	}
}

func sameRecord(names1, names2 []string) bool {
	if len(names1) != len(names2) {
		return false
	}
	for i := range names1 {
		if names1[i] != names2[i] {
			return false
		}
	}
	return true
}

// CreateCut returns a new record value derived by keeping only the fields
// specified by name in the fields slice.
func (c *Cut) cut(in *zson.Record) (*zson.Record, error) {
	// Check if we already have an output descriptor for this
	// input type
	d, ok := c.cutmap[in.ID]
	if ok && d == nil {
		// One or more cut fields isn't present in this type of
		// input record, drop it now.
		return nil, nil
	}

	c.builder.Reset()
	var types []zeek.Type
	if d == nil {
		types = make([]zeek.Type, 0, len(c.fields))
	}
	// Build the output record.  If we've already seen this input
	// record type, we don't care about the types, but if we haven't
	// gather the types as well so we can construct the output
	// descriptor.
	for _, field := range c.fields {
		val := field.resolver(in)
		if d == nil {
			if val.Type == nil {
				// a field is missing... block this descriptor
				c.cutmap[in.ID] = nil
				c.nblocked++
				return nil, nil
			}
			types = append(types, val.Type)
		}
		for range field.containerBegins {
			c.builder.BeginContainer()
		}
		c.builder.Append(val.Body)
		for i := 0; i < field.containerEnds; i++ {
			c.builder.EndContainer()
		}
	}
	if d == nil {
		d = c.getOutputDescriptor(types)
		c.cutmap[in.ID] = d
	}

	return zson.NewRecordNoTs(d, c.builder.Encode()), nil
}

// Using similar logic to the main loop inside cut(), allocate a
// descriptor for an output record in which the cut fields have the types
// indicated in the passed-in array of types.
func (c *Cut) getOutputDescriptor(types []zeek.Type) *zson.Descriptor {
	type rec struct {
		name string
		cols []zeek.Column
	}
	current := &rec{"", nil}
	stack := make([]*rec, 1)
	stack[0] = current

	for i, field := range c.fields {
		for _, name := range field.containerBegins {
			current = &rec{name, nil}
			stack = append(stack, current)
		}

		current.cols = append(current.cols, zeek.Column{Name: field.name, Type: types[i]})

		for j := 0; j < field.containerEnds; j++ {
			recType := zeek.LookupTypeRecord(current.cols)
			slen := len(stack)
			stack = stack[:slen-1]
			cur := stack[slen-2]
			cur.cols = append(cur.cols, zeek.Column{Name: current.name, Type: recType})
			current = cur
		}
	}
	if len(stack) != 1 {
		panic("Mismatched container begin/end")
	}
	return c.Resolver.GetByColumns(stack[0].cols)
}

func (c *Cut) warn() {
	if len(c.cutmap) > c.nblocked {
		return
	}
	var msg string
	if len(c.fields) == 1 {
		msg = fmt.Sprintf("Cut field %s not present in input", c.fields[0].fullname)
	} else {
		names := make([]string, 0, len(c.fields))
		for _, f := range c.fields {
			names = append(names, f.fullname)
		}
		msg = fmt.Sprintf("Cut fields %s not present together in input", strings.Join(names, ","))
	}
	c.Warnings <- msg
}

func (c *Cut) Pull() (zson.Batch, error) {
	batch, err := c.Get()
	if EOS(batch, err) {
		c.warn()
		return nil, err
	}
	defer batch.Unref()
	//
	// Make new records with only the fields specified.
	// If a field specified doesn't exist, we don't include that record.
	// if the types change for the fields specified, we drop those records.
	//
	recs := make([]*zson.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		out, err := c.cut(in)
		if err != nil {
			return nil, err
		}
		if out != nil {
			recs = append(recs, out)
		}
	}
	if len(recs) == 0 {
		c.warn()
		return nil, nil
	}
	return zson.NewArray(recs, batch.Span()), nil
}
