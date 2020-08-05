package archive

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
)

const zarExt = ".zar"

// XXX Embedding the type and field names like this can result in some clunky
// file names. We might want to re-work the naming scheme.

func typeZdxName(t zng.Type) string {
	return "zdx-type-" + t.String()
}

func fieldZdxName(fieldname string) string {
	return "zdx-field-" + fieldname
}

func IndexDirTree(ark *Archive, rules []Rule, path string, progress chan<- string) error {
	err := Walk(ark, func(zardir iosrc.URI) error {
		logPath := Localize(zardir, path)
		return run(zardir, rules, logPath, progress)
	})
	if err != nil {
		return err
	}
	var infos []IndexInfo
	for _, r := range rules {
		infos = append(infos, IndexInfo{r.typ, r.path})
	}
	return ark.AddIndexes(infos)
}

func runOne(zardir iosrc.URI, rule Rule, inputPath iosrc.URI, progress chan<- string) error {
	rc, err := iosrc.NewReader(inputPath)
	if err != nil {
		return err
	}
	defer rc.Close()
	zctx := resolver.NewContext()
	r := zngio.NewReader(rc, zctx)
	fgi, err := NewFlowgraphIndexer(zctx, rule.Path(zardir), rule.keys, rule.framesize)
	if err != nil {
		return err
	}
	defer fgi.Close()
	out, err := driver.Compile(context.TODO(), rule.proc, zctx, []zbuf.Reader{r}, driver.Config{
		Custom: &compiler{},
	})
	if err != nil {
		return err
	}
	if progress != nil {
		progress <- fmt.Sprintf("%s: creating index %s", inputPath, rule.Path(zardir))
	}
	return driver.Run(out, fgi, nil)
}

func run(zardir iosrc.URI, rules []Rule, logPath iosrc.URI, progress chan<- string) error {
	for _, rule := range rules {
		if err := runOne(zardir, rule, logPath, progress); err != nil {
			return err
		}
	}
	return nil
}

type FlowgraphIndexer struct {
	zctx    *resolver.Context
	w       *zdx.Writer
	keyType zng.Type
	cutter  *proc.Cutter
}

func NewFlowgraphIndexer(zctx *resolver.Context, uri iosrc.URI, keys []string, framesize int) (*FlowgraphIndexer, error) {
	if len(keys) == 0 {
		keys = []string{keyName}
	}
	writer, err := zdx.NewWriter(zctx, uri.String(), keys, framesize)
	if err != nil {
		return nil, err
	}
	return &FlowgraphIndexer{
		zctx:   zctx,
		w:      writer,
		cutter: proc.NewStrictCutter(zctx, false, keys, keys),
	}, nil
}

func (f *FlowgraphIndexer) Write(_ int, batch zbuf.Batch) error {
	for i := 0; i < batch.Length(); i++ {
		rec := batch.Index(i)
		key, err := f.cutter.Cut(rec)
		if err != nil {
			return fmt.Errorf("checking index record: %w", err)
		}
		if f.keyType == nil {
			f.keyType = key.Type
		}
		if key.Type.ID() != f.keyType.ID() {
			return fmt.Errorf("key type changed from %s to %s", f.keyType, key.Type)
		}
		if err := f.w.Write(rec); err != nil {
			return err
		}
	}
	batch.Unref()
	return nil
}

func (f *FlowgraphIndexer) Close() error {
	return f.w.Close()
}

func (f *FlowgraphIndexer) Warn(warning string) error          { return nil }
func (f *FlowgraphIndexer) Stats(stats api.ScannerStats) error { return nil }
func (f *FlowgraphIndexer) ChannelEnd(cid int) error           { return nil }
