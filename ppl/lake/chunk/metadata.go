package chunk

import (
	"bytes"
	"context"
	"fmt"

	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/segmentio/ksuid"
)

type Metadata struct {
	First       nano.Ts
	Last        nano.Ts
	RecordCount uint64
	Masks       []ksuid.KSUID
	Size        int64
}

func UnmarshalMetadata(b []byte, order zbuf.Order) (Metadata, error) {
	zctx := resolver.NewContext()
	zr := zngio.NewReader(bytes.NewReader(b), zctx)
	rec, err := zr.Read()
	if err != nil {
		return Metadata{}, err
	}
	var md Metadata
	if err := resolver.UnmarshalRecord(rec, &md); err != nil {
		return Metadata{}, err
	}
	if err := mdTsOrderCheck("read", order, md.First, md.Last); err != nil {
		return Metadata{}, err
	}
	return md, nil
}

func (m Metadata) Chunk(dir iosrc.URI, id ksuid.KSUID) Chunk {
	return Chunk{
		Dir:         dir,
		Id:          id,
		First:       m.First,
		Last:        m.Last,
		RecordCount: m.RecordCount,
		Masks:       m.Masks,
		Size:        m.Size,
	}
}

func (m Metadata) Write(ctx context.Context, uri iosrc.URI, order zbuf.Order) error {
	if err := mdTsOrderCheck("write", order, m.First, m.Last); err != nil {
		return err
	}
	rec, err := resolver.NewMarshaler().MarshalRecord(m)
	if err != nil {
		return err
	}
	out, err := iosrc.NewWriter(ctx, uri)
	if err != nil {
		return err
	}
	zw := zngio.NewWriter(bufwriter.New(out), zngio.WriterOpts{})
	if err := zw.Write(rec); err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}

func MetadataPath(dir iosrc.URI, id ksuid.KSUID) iosrc.URI {
	return dir.AppendPath(fmt.Sprintf("%s-%s.zng", FileKindMetadata, id))
}

func mdTsOrderCheck(op string, order zbuf.Order, first, last nano.Ts) error {
	x, y := first, last
	if order == zbuf.OrderDesc {
		x, y = y, x
	}
	if x <= y {
		return nil
	}
	return fmt.Errorf("metadata failed order check op %s order %s first %d last %d", op, order, first, last)
}
