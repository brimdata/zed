package lakeio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/charm"
	"github.com/brimdata/zed/pkg/terminal/color"
	"github.com/brimdata/zed/pkg/units"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

type Writer struct {
	writer   io.WriteCloser
	zson     *zson.Formatter
	commits  table
	rulename string
	width    int
	colors   color.Stack
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{
		writer:  w,
		zson:    zson.NewFormatter(0),
		commits: make(table),
		width:   80, //XXX
	}
}

func (w *Writer) Write(rec *zng.Record) error {
	var v interface{}
	if err := unmarshaler.Unmarshal(rec.Value, &v); err != nil {
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

func (w *Writer) WriteZSON(rec *zng.Record) error {
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
	case *lake.PoolConfig:
		formatPoolConfig(b, v)
	case segment.Reference:
		formatSegment(b, &v, "", 0)
	case *segment.Reference:
		formatSegment(b, v, "", 0)
	case lake.Partition:
		formatPartition(b, v)
	case *actions.CommitMessage:
		t.formatCommit(b, v, width, colors)
	case *actions.StagedCommit:
		t.formatStaged(b, v, colors)
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
	default:
		if action, ok := v.(actions.Interface); ok {
			t.append(action)
			return
		}
		b.WriteString(fmt.Sprintf("lake format: unknown type: %T\n", v))
	}
}

func formatCommit(b *bytes.Buffer, txn *commit.Transaction) {
	b.WriteString(fmt.Sprintf("commit %s\n", txn.ID))
	for _, action := range txn.Actions {
		b.WriteString(fmt.Sprintf("  segment %s\n", action))
	}
}

func formatPoolConfig(b *bytes.Buffer, p *lake.PoolConfig) {
	b.WriteString(p.Name)
	b.WriteString(" ")
	b.WriteString(p.ID.String())
	b.WriteString(" key ")
	b.WriteString(field.List(p.Layout.Keys).String())
	b.WriteString(" order ")
	b.WriteString(p.Layout.Order.String())
	b.WriteByte('\n')
}

func tab(b *bytes.Buffer, indent int) {
	for k := 0; k < indent; k++ {
		b.WriteByte(' ')
	}
}

func formatSegment(b *bytes.Buffer, seg *segment.Reference, prefix string, indent int) {
	tab(b, indent)
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteByte(' ')
	}
	b.WriteString(seg.ID.String())
	objectSize := units.Bytes(seg.RowSize).Abbrev()
	b.WriteString(fmt.Sprintf(" %s bytes %d records", objectSize, seg.Count))
	b.WriteString("\n  ")
	tab(b, indent)
	b.WriteString(" from ")
	b.WriteString(zson.String(seg.First))
	b.WriteString(" to ")
	b.WriteString(zson.String(seg.Last))
	b.WriteByte('\n')
}

func formatPartition(b *bytes.Buffer, p lake.Partition) {
	b.WriteString("from ")
	b.WriteString(zson.String(p.First()))
	b.WriteString(" to ")
	b.WriteString(zson.String(p.Last()))
	b.WriteByte('\n')
	for _, seg := range p.Segments {
		formatSegment(b, seg, "", 2)
	}
}

type table map[ksuid.KSUID][]actions.Interface

func (t table) append(a actions.Interface) {
	id := a.CommitID()
	t[id] = append(t[id], a)
}

func (t table) formatStaged(b *bytes.Buffer, commit *actions.StagedCommit, colors *color.Stack) {
	id := commit.CommitID()
	colors.Start(b, color.GrayYellow)
	b.WriteString("staged ")
	b.WriteString(id.String())
	colors.End(b)
	b.WriteString("\n\n")
	t.formatActions(b, id)
}

func (t table) formatCommit(b *bytes.Buffer, commit *actions.CommitMessage, width int, colors *color.Stack) {
	id := commit.CommitID()
	colors.Start(b, color.GrayYellow)
	b.WriteString("commit ")
	b.WriteString(id.String())
	colors.End(b)
	b.WriteString("\nAuthor: ")
	b.WriteString(commit.Author)
	b.WriteString("\nDate:   ")
	b.WriteString(commit.Date.String())
	b.WriteString("\n\n")
	if commit.Message != "" {
		b.WriteString(charm.FormatParagraph(commit.Message, "    ", width))
	}
	t.formatActions(b, id)
}

func (t table) formatActions(b *bytes.Buffer, id ksuid.KSUID) {
	for _, action := range t[id] {
		switch action := action.(type) {
		case *actions.Add:
			formatAdd(b, 4, action)
		case *actions.AddIndex:
			formatAddIndex(b, 4, action)
		case *actions.Delete:
			formatDelete(b, 4, action)
		}
	}
	b.WriteString("\n")
}

func formatDelete(b *bytes.Buffer, indent int, delete *actions.Delete) {
	tab(b, indent)
	b.WriteString("Delete ")
	b.WriteString(delete.ID.String())
	b.WriteByte('\n')
}

func formatAdd(b *bytes.Buffer, indent int, add *actions.Add) {
	formatSegment(b, &add.Segment, "Add", indent)
}

func formatAddIndex(b *bytes.Buffer, indent int, addx *actions.AddIndex) {
	formatIndexObject(b, &addx.Index, "AddIndex", indent)
}

func formatIndexObject(b *bytes.Buffer, index *index.Reference, prefix string, indent int) {
	tab(b, indent)
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteByte(' ')
	}
	b.WriteString(fmt.Sprintf("%s index %s segment", index.Rule.RuleID(), index.SegmentID))
	b.WriteByte('\n')
}
