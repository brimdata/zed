package index

import (
	"fmt"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/segmentio/ksuid"
)

type Object struct {
	Rule Rule
	ID   ksuid.KSUID
}

func (o Object) String() string {
	//XXX data object looks like this:
	//	return fmt.Sprintf("%s %d record%s in %d data bytes", o.ID, o.Count, plural(int(o.Count)), o.RowSize)
	return fmt.Sprintf("%s/%s", o.Rule.RuleID(), o.ID)
}

func (o Object) Path(path *storage.URI) *storage.URI {
	return path.AppendPath(o.Rule.RuleID().String())
	//	return xObjectPath(path, o.Rule, o.ID)
}
