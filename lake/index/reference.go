package index

import (
	"fmt"

	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/segmentio/ksuid"
)

type Reference struct {
	Rule      Rule
	SegmentID ksuid.KSUID
}

func (r Reference) String() string {
	return fmt.Sprintf("%s/%s", r.Rule.ID, r.SegmentID)
}

func (r Reference) ObjectName() string {
	return ObjectName(r.SegmentID)
}

func ObjectName(id ksuid.KSUID) string {
	return fmt.Sprintf("idx-%s.zng", id)
}

func (r Reference) ObjectDir(path iosrc.URI) iosrc.URI {
	return ObjectDir(path, r.Rule)
}

func ObjectDir(path iosrc.URI, rule Rule) iosrc.URI {
	return path.AppendPath(rule.ID.String())
}

func (r Reference) ObjectPath(path iosrc.URI) iosrc.URI {
	return ObjectPath(path, r.Rule, r.SegmentID)
}

func ObjectPath(path iosrc.URI, rule Rule, id ksuid.KSUID) iosrc.URI {
	return ObjectDir(path, rule).AppendPath(ObjectName(id))
}

// func (i Index) Filename() string {
// return indexFilename(i.Definition.ID)
// }

// func RemoveIndices(ctx context.Context, dir iosrc.URI, defs []*Definition) ([]Index, error) {
// removed := make([]Index, 0, len(defs))
// for _, def := range defs {
// path := IndexPath(dir, def.ID)
// if err := iosrc.Remove(ctx, path); err != nil {
// if zqe.IsNotFound(err) {
// continue
// }
// return nil, err
// }
// removed = append(removed, Index{def, dir})
// }
// return removed, nil
// }

// func Find(ctx context.Context, zctx *zson.Context, path iosrc.URI, id ksuid.KSUID, patterns ...string) (zbuf.ReadCloser, error) {
// return FindFromPath(ctx, zctx, ObjectPath(d, id), patterns...)
// }

// func FindFromPath(ctx context.Context, zctx *zson.Context, idxfile iosrc.URI, patterns ...string) (zbuf.ReadCloser, error) {
// finder, err := index.NewFinderReader(ctx, zctx, idxfile, patterns...)
// if err != nil {
// return nil, fmt.Errorf("index find %s: %w", idxfile, err)
// }
// return finder, nil
// }

// var indexFileRegex = regexp.MustCompile(`idx-([0-9A-Za-z]{27}).zng$`)

// func parseIndexFile(name string) (ksuid.KSUID, error) {
// match := indexFileRegex.FindStringSubmatch(name)
// if match == nil {
// return ksuid.Nil, fmt.Errorf("invalid index file: %s", name)
// }
// return ksuid.Parse(match[1])
// }

// func mkdir(d iosrc.URI) error {
// return iosrc.MkdirAll(d, 0700)
// }

// func infos(ctx context.Context, d iosrc.URI) ([]iosrc.Info, error) {
// infos, err := iosrc.ReadDir(ctx, d)
// if zqe.IsNotFound(err) {
// return nil, mkdir(d)
// }
// return infos, err
// }
