package seekindex

import (
	"context"

	"github.com/brimsec/zq/microindex"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var schema = []zng.Column{
	{"ts", zng.TypeTime},
	{"offset", zng.TypeInt64},
}

type Builder struct {
	builder *zng.Builder
	writer  *microindex.Writer
}

func NewBuilder(ctx context.Context, path string, order zbuf.Order) (*Builder, error) {
	zctx := resolver.NewContext()
	writer, err := microindex.NewWriterWithContext(ctx, zctx, path, microindex.Order(order), microindex.Keys("ts"))
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
