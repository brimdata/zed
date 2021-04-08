package seekindex

import (
	"context"

	"github.com/brimdata/zed/index"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

var schema = []zng.Column{
	{"ts", zng.TypeTime},
	{"offset", zng.TypeInt64},
}

type Builder struct {
	builder *zng.Builder
	writer  *index.Writer
}

func NewBuilder(ctx context.Context, path string, order zbuf.Order) (*Builder, error) {
	zctx := zson.NewContext()
	writer, err := index.NewWriterWithContext(ctx, zctx, path, index.Order(order), index.Keys("ts"))
	if err != nil {
		return nil, err
	}
	return &Builder{
		builder: zng.NewBuilder(zctx.MustLookupTypeRecord(schema)),
		writer:  writer,
	}, nil
}

func (b *Builder) Enter(ts nano.Ts, offset int64) error {
	rec := b.builder.Build(zng.EncodeTime(ts), zng.EncodeInt(offset))
	return b.writer.Write(rec)
}

func (b *Builder) Abort() error {
	return b.writer.Abort()
}

// Close closes the underlying microindex.Writer. Should an error occur the
// microindex will be deleted via a call to Abort.
func (b *Builder) Close() error {
	err := b.writer.Close()
	if err != nil {
		b.Abort()
	}
	return err
}
