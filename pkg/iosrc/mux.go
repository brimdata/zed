package iosrc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

// MuxSource is a Source that routes each function call to the correct Source
// based off the provided URI's scheme.
type MuxSource struct {
	schemes map[string]Source
}

func NewMuxSource(m map[string]Source) *MuxSource {
	return &MuxSource{m}
}

func (m *MuxSource) AddScheme(scheme string, source Source) {
	m.schemes[scheme] = source
}

func getScheme(uri URI) string {
	if uri.Scheme == "" {
		return FileScheme
	}
	return uri.Scheme
}

func (m *MuxSource) GetSource(u URI) (Source, error) {
	scheme := getScheme(u)
	source, ok := m.schemes[scheme]
	if !ok {
		return nil, fmt.Errorf("unknown scheme: %q", scheme)
	}
	return source, nil
}

func (m *MuxSource) NewReader(ctx context.Context, uri URI) (Reader, error) {
	source, err := m.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.NewReader(ctx, uri)
}

func (m *MuxSource) NewWriter(ctx context.Context, uri URI) (io.WriteCloser, error) {
	source, err := m.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.NewWriter(ctx, uri)
}

func (m *MuxSource) ReadFile(ctx context.Context, uri URI) ([]byte, error) {
	source, err := m.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.ReadFile(ctx, uri)
}

func (m *MuxSource) WriteFile(ctx context.Context, uri URI, d []byte) error {
	source, err := m.GetSource(uri)
	if err != nil {
		return err
	}
	return source.WriteFile(ctx, d, uri)
}

func (m *MuxSource) WriteFileIfNotExists(ctx context.Context, uri URI, d []byte) error {
	source, err := m.GetSource(uri)
	if err != nil {
		return err
	}
	return source.WriteFileIfNotExists(ctx, d, uri)
}

func (m *MuxSource) Exists(ctx context.Context, uri URI) (bool, error) {
	source, err := m.GetSource(uri)
	if err != nil {
		return false, err
	}
	return source.Exists(ctx, uri)
}

func (m *MuxSource) Remove(ctx context.Context, uri URI) error {
	source, err := m.GetSource(uri)
	if err != nil {
		return err
	}
	return source.Remove(ctx, uri)
}

func (m *MuxSource) RemoveAll(ctx context.Context, uri URI) error {
	source, err := m.GetSource(uri)
	if err != nil {
		return err
	}
	return source.RemoveAll(ctx, uri)
}

func (m *MuxSource) Stat(ctx context.Context, uri URI) (Info, error) {
	source, err := m.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.Stat(ctx, uri)
}

func (m *MuxSource) ReadDir(ctx context.Context, uri URI) ([]Info, error) {
	source, err := m.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.ReadDir(ctx, uri)
}

func (m *MuxSource) Replace(ctx context.Context, uri URI, fn func(w io.Writer) error) error {
	src, err := m.GetSource(uri)
	if err != nil {
		return err
	}
	replacerAble, ok := src.(ReplacerAble)
	if !ok {
		return errors.New("source does not support replacement")
	}
	r, err := replacerAble.NewReplacer(ctx, uri)
	if err != nil {
		return err
	}
	if err := fn(r); err != nil {
		r.Abort()
		return err
	}
	return r.Close()
}

// MkdirAll will run Source.MkdirAll on the provided URI if the URI's source
// is a DirMaker, otherwise it will do nothing.
func (m *MuxSource) MkdirAll(uri URI, mode os.FileMode) error {
	src, err := m.GetSource(uri)
	if err != nil {
		return err
	}
	if mkr, ok := src.(DirMaker); ok {
		err = mkr.MkdirAll(uri, mode)
	}
	return err
}
