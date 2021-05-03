package index

import (
	"fmt"

	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/segmentio/ksuid"
)

type Reference struct {
	Index     Index
	SegmentID ksuid.KSUID
}

func (r Reference) String() string {
	return fmt.Sprintf("%s/%s", r.Index.ID, r.SegmentID)
}

func (r Reference) ObjectName() string {
	return ObjectName(r.SegmentID)
}

func ObjectName(id ksuid.KSUID) string {
	return fmt.Sprintf("%s.zng", id)
}

func (r Reference) ObjectDir(path iosrc.URI) iosrc.URI {
	return ObjectDir(path, r.Index)
}

func ObjectDir(path iosrc.URI, rule Index) iosrc.URI {
	return path.AppendPath(rule.ID.String())
}

func (r Reference) ObjectPath(path iosrc.URI) iosrc.URI {
	return ObjectPath(path, r.Index, r.SegmentID)
}

func ObjectPath(path iosrc.URI, rule Index, id ksuid.KSUID) iosrc.URI {
	return ObjectDir(path, rule).AppendPath(ObjectName(id))
}
