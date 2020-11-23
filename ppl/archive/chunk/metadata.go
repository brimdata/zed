package chunk

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/segmentio/ksuid"
)

type Metadata struct {
	First       nano.Ts
	Last        nano.Ts
	RecordCount uint64
	Masks       []ksuid.KSUID
	Size        int64
}

func (md Metadata) Chunk(dir iosrc.URI, id ksuid.KSUID) Chunk {
	return Chunk{
		Dir:         dir,
		Id:          id,
		First:       md.First,
		Last:        md.Last,
		RecordCount: md.RecordCount,
		Masks:       md.Masks,
		Size:        md.Size,
	}
}

func MetadataPath(dir iosrc.URI, id ksuid.KSUID) iosrc.URI {
	return dir.AppendPath(fmt.Sprintf("%s-%s.zng", FileKindMetadata, id))
}

func WriteMetadata(ctx context.Context, uri iosrc.URI, md Metadata) error {
	zctx := resolver.NewContext()
	rec, err := resolver.MarshalRecord(zctx, md)
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

func ReadMetadata(ctx context.Context, uri iosrc.URI) (Metadata, error) {
	in, err := iosrc.NewReader(ctx, uri)
	if err != nil {
		return Metadata{}, err
	}
	defer in.Close()
	zctx := resolver.NewContext()
	zr := zngio.NewReader(in, zctx)
	rec, err := zr.Read()
	if err != nil {
		return Metadata{}, err
	}
	var md Metadata
	if err := resolver.UnmarshalRecord(zctx, rec, &md); err != nil {
		return Metadata{}, err
	}
	return md, nil
}
