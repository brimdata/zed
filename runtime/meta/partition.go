package meta

import (
	"fmt"

	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zson"
)

// A Partition is a logical view of the records within a time span, stored
// in one or more data objects.  This provides a way to return the list of
// objects that should be scanned along with a span to limit the scan
// to only the span involved.
type Partition struct {
	extent.Span
	Compare expr.CompareFn
	Objects []*data.ObjectScan
}

func (p Partition) IsZero() bool {
	return p.Objects == nil
}

func (p Partition) FormatRangeOf(index int) string {
	o := p.Objects[index]
	return fmt.Sprintf("[%s-%s,%s-%s]", zson.String(*p.First()), zson.String(*p.Last()), zson.String(o.First), zson.String(o.Last))
}

func (p Partition) FormatRange() string {
	return fmt.Sprintf("[%s-%s]", zson.String(*p.First()), zson.String(*p.Last()))
}
