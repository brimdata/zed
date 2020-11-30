package index

import (
	"context"

	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
	"github.com/segmentio/ksuid"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

type Dir iosrc.URI

func (d Dir) ensureDir(ctx context.Context) error {
	return iosrc.MkdirAll(iosrc.URI(d), 0755)
}

func (d Dir) infos(ctx context.Context) ([]iosrc.Info, error) {
	infos, err := iosrc.ReadDir(ctx, iosrc.URI(d))
	if zqe.IsNotFound(err) {
		if err := iosrc.MkdirAll(iosrc.URI(d), 0755); err != nil {
			return nil, err
		}
		err = nil
	}
	return infos, err
}

func (d Dir) Map(ctx context.Context) (map[ksuid.KSUID]bool, error) {
	infos, err := d.infos(ctx)
	if err != nil {
		return nil, err
	}
	indices := make(map[ksuid.KSUID]bool)
	for _, info := range infos {
		uuid, err := parseIndexFile(info.Name())
		if err != nil {
			return nil, err
		}
		indices[uuid] = false
	}
	return indices, nil
}

func (d Dir) List(ctx context.Context) ([]ksuid.KSUID, error) {
	infos, err := d.infos(ctx)
	if err != nil {
		return nil, err
	}
	var indices []ksuid.KSUID
	for _, info := range infos {
		uuid, err := parseIndexFile(info.Name())
		if err != nil {
			return nil, err
		}
		indices = append(indices, uuid)
	}
	return indices, nil
}

func (d Dir) Sync(ctx context.Context, r zbuf.Scanner, defs []*Def) error {
	indices, err := d.Map(ctx)
	if err != nil {
		return err
	}
	m := make(map[ksuid.KSUID]*Def)
	var add, remove []*Def
	for _, def := range defs {
		m[def.ID] = def
		if _, ok := indices[def.ID]; !ok {
			add = append(add, def)
		}
	}
	for id := range indices {
		if def, ok := m[id]; !ok {
			remove = append(remove, def)
		}
	}
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return d.Add(ctx, r, add...)
	})
	group.Go(func() error {
		return d.Remove(ctx, remove...)
	})
	return group.Wait()
}

func (d Dir) Index(id ksuid.KSUID) iosrc.URI {
	return IndexPath(iosrc.URI(d), id)
}

func (d Dir) Find(ctx context.Context, id ksuid.KSUID, patterns ...string) (*zng.Record, error) {
	return Find(ctx, d.Index(id), patterns...)
}

func (d Dir) FindAll(ctx context.Context, id ksuid.KSUID, patterns ...string) (zbuf.Batch, error) {
	return FindAll(ctx, d.Index(id), patterns...)
}

func (d Dir) NewWriter(ctx context.Context, defs ...*Def) (DirWriter, error) {
	return NewDirWriter(ctx, d, defs)
}

func (d Dir) newIndexWriter(ctx context.Context, def *Def) (*Writer, error) {
	u := d.Index(def.ID)
	w, err := NewWriter(ctx, u, def)
	if zqe.IsNotFound(err) {
		if err := d.ensureDir(ctx); err != nil {
			return nil, err
		}
		err = nil
		w, err = NewWriter(ctx, u, def)
	}
	return w, err
}

func (d Dir) Add(ctx context.Context, p zbuf.Puller, defs ...*Def) error {
	writers, err := NewDirWriter(ctx, d, defs)
	if err != nil {
		return err
	}
	if err := zbuf.CopyBatchesWithContext(ctx, writers, p); err != nil {
		writers.Abort()
		return err
	}
	return writers.Close()
}

func (d Dir) AddFromPath(ctx context.Context, file iosrc.URI, defs ...*Def) error {
	writers, err := NewDirWriter(ctx, d, defs)
	if err != nil {
		return err
	}
	if len(writers) == 0 {
		return nil
	}
	r, err := iosrc.NewReader(ctx, file)
	if err != nil {
		writers.Abort()
		return nil
	}
	defer r.Close()
	s, err := zngio.NewReader(r, resolver.NewContext()).NewScanner(ctx, nil, nano.MaxSpan)
	if err != nil {
		writers.Abort()
		return nil
	}
	if err := zbuf.CopyBatchesWithContext(ctx, writers, s); err != nil {
		writers.Abort()
		return err
	}
	return writers.Close()
}

func (d Dir) Remove(ctx context.Context, defs ...*Def) error {
	var merr error
	for _, def := range defs {
		u := d.Index(def.ID)
		err := iosrc.Remove(ctx, u)
		if err != nil && !zqe.IsNotFound(err) {
			merr = multierr.Append(merr, err)
		}
	}
	return merr
}
