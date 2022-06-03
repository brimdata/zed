package lakeio

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/terminal/color"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type WriterOpts struct {
	Head string
}

type Writer struct {
	writer   io.WriteCloser
	zson     *zson.Formatter
	commits  table
	branches map[ksuid.KSUID][]string
	rulename string
	width    int
	colors   color.Stack
	headID   ksuid.KSUID
	headName string
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	writer := &Writer{
		writer:   w,
		zson:     zson.NewFormatter(0, nil),
		commits:  make(table),
		branches: make(map[ksuid.KSUID][]string),
		width:    80, //XXX
	}
	// If head is an ID, we assume its detached and format accordingly.
	// If it's name, we'll print "HEAD -> branch" in the branch name listing
	// if we encounter that name.
	if headID, err := lakeparse.ParseID(opts.Head); err == nil {
		writer.headID = headID
	} else {
		writer.headName = opts.Head
	}
	return writer
}

func (w *Writer) Write(rec *zed.Value) error {
	var v interface{}
	if err := unmarshaler.Unmarshal(rec, &v); err != nil {
		return w.WriteZSON(rec)
	}
	var b bytes.Buffer
	w.formatValue(w.commits, &b, v, w.width, &w.colors)
	_, err := w.writer.Write(b.Bytes())
	return err
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) WriteZSON(rec *zed.Value) error {
	s, err := w.zson.FormatRecord(rec)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w.writer, s); err != nil {
		return err
	}
	_, err = io.WriteString(w.writer, "\n")
	return err
}

func (w *Writer) formatValue(t table, b *bytes.Buffer, v interface{}, width int, colors *color.Stack) {
	switch v := v.(type) {
	case *pools.Config:
		formatPoolConfig(b, v)
	case *lake.BranchMeta:
		formatBranchMeta(b, v, width, w.headID, w.headName, colors)
	case data.Object:
		formatDataObject(b, &v, "", 0)
	case *data.Object:
		formatDataObject(b, v, "", 0)
	case lake.Partition:
		formatPartition(b, v)
	case *commits.Commit:
		branches := w.branches[v.ID]
		t.formatCommit(b, v, branches, w.headName, w.headID, width, colors)
	case index.Rule:
		name := v.RuleName()
		if name != w.rulename {
			w.rulename = name
			b.WriteString(name)
			b.WriteByte('\n')
		}
		tab(b, 4)
		b.WriteString(v.String())
		b.WriteByte('\n')
	case *lake.BranchTip:
		w.branches[v.Commit] = append(w.branches[v.Commit], v.Name)
	default:
		if action, ok := v.(commits.Action); ok {
			t.append(action)
			return
		}
		b.WriteString(fmt.Sprintf("lake format: unknown type: %T\n", v))
	}
}

func formatPoolConfig(b *bytes.Buffer, p *pools.Config) {
	b.WriteString(p.Name)
	b.WriteByte(' ')
	b.WriteString(p.ID.String())
	b.WriteString(" key ")
	b.WriteString(field.List(p.Layout.Keys).String())
	b.WriteString(" order ")
	b.WriteString(p.Layout.Order.String())
	b.WriteByte('\n')
}

func formatBranchMeta(b *bytes.Buffer, p *lake.BranchMeta, width int, headID ksuid.KSUID, headName string, colors *color.Stack) {
	b.WriteString(p.Pool.Name)
	b.WriteByte('@')
	b.WriteString(p.Branch.Name)
	b.WriteByte(' ')
	colors.Start(b, color.GrayYellow)
	b.WriteString("commit ")
	b.WriteString(p.Branch.Commit.String())
	if headID == p.Branch.Commit || headName == p.Branch.Name {
		b.WriteString(" (")
		colors.Start(b, color.Turqoise)
		b.WriteString(color.Embolden("HEAD"))
		colors.End(b)
		b.WriteByte(')')
	}
	colors.End(b)
	b.WriteByte('\n')
}

func tab(b *bytes.Buffer, indent int) {
	for k := 0; k < indent; k++ {
		b.WriteByte(' ')
	}
}

func formatDataObject(b *bytes.Buffer, object *data.Object, prefix string, indent int) {
	tab(b, indent)
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteByte(' ')
	}
	b.WriteString(object.ID.String())
	objectSize := units.Bytes(object.Size).Abbrev()
	b.WriteString(fmt.Sprintf(" %s bytes %d records", objectSize, object.Count))
	b.WriteString("\n  ")
	tab(b, indent)
	b.WriteString(" from ")
	b.WriteString(zson.String(object.First))
	b.WriteString(" to ")
	b.WriteString(zson.String(object.Last))
	b.WriteByte('\n')
}

func formatPartition(b *bytes.Buffer, p lake.Partition) {
	b.WriteString("from ")
	b.WriteString(zson.String(*p.First()))
	b.WriteString(" to ")
	b.WriteString(zson.String(*p.Last()))
	b.WriteByte('\n')
	for _, o := range p.Objects {
		formatDataObject(b, o, "", 2)
	}
}

type table map[ksuid.KSUID][]commits.Action

func (t table) append(a commits.Action) {
	id := a.CommitID()
	t[id] = append(t[id], a)
}

func (t table) formatCommit(b *bytes.Buffer, commit *commits.Commit, branches []string, headName string, headID ksuid.KSUID, width int, colors *color.Stack) {
	id := commit.CommitID()
	colors.Start(b, color.GrayYellow)
	b.WriteString("commit ")
	b.WriteString(id.String())
	if len(branches) > 0 {
		b.WriteString(" (")
		for k, name := range branches {
			if k != 0 {
				b.WriteString(", ")
			}
			if name == headName {
				colors.Start(b, color.Turqoise)
				b.WriteString(color.Embolden("HEAD -> "))
				colors.End(b)
			}
			colors.Start(b, color.Green)
			b.WriteString(name)
			colors.End(b)
		}
		b.WriteString(")")
	} else if commit.ID == headID {
		b.WriteString(" (")
		colors.Start(b, color.Blue)
		b.WriteString("HEAD")
		colors.End(b)
		b.WriteString(")")
	}
	colors.End(b)
	b.WriteString("\nAuthor: ")
	b.WriteString(commit.Author)
	b.WriteString("\nDate:   ")
	b.WriteString(commit.Date.String())
	b.WriteString("\n\n")
	if commit.Message != "" {
		s := charm.FormatParagraph(commit.Message, "    ", width)
		s = strings.TrimRight(s, " \n") + "\n\n"
		b.WriteString(s)
	}
}

func (t table) formatActions(b *bytes.Buffer, id ksuid.KSUID) {
	for _, action := range t[id] {
		switch action := action.(type) {
		case *commits.Add:
			formatAdd(b, 4, action)
		case *commits.AddIndex:
			formatAddIndex(b, 4, action)
		case *commits.Delete:
			formatDelete(b, 4, action)
		case *commits.DeleteIndex:
			formatDeleteIndex(b, 4, action)
		}
	}
	b.WriteString("\n")
}

func formatDelete(b *bytes.Buffer, indent int, delete *commits.Delete) {
	tab(b, indent)
	b.WriteString("Delete ")
	b.WriteString(delete.ID.String())
	b.WriteByte('\n')
}

func formatAdd(b *bytes.Buffer, indent int, add *commits.Add) {
	formatDataObject(b, &add.Object, "Add", indent)
}

func formatAddIndex(b *bytes.Buffer, indent int, addx *commits.AddIndex) {
	formatIndexObject(b, addx.Object.Rule.RuleID(), addx.Object.ID, "AddIndex", indent)
}

func formatDeleteIndex(b *bytes.Buffer, indent int, delx *commits.DeleteIndex) {
	formatIndexObject(b, delx.RuleID, delx.ID, "DeleteIndex", indent)
}

func formatIndexObject(b *bytes.Buffer, ruleID, id ksuid.KSUID, prefix string, indent int) {
	tab(b, indent)
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteByte(' ')
	}
	fmt.Fprintf(b, "%s index %s object", ruleID, id) //XXX
	b.WriteByte('\n')
}
