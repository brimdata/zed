//go:generate mockgen -destination=./mock/mock_source.go -package=mock github.com/brimdata/zed/pkg/iosrc Source

package iosrc

import (
	"context"
	"io"
	"os"
	"time"
)

const FileScheme = "file"

var DefaultMuxSource = NewMuxSource(map[string]Source{
	"file":  DefaultFileSource,
	"stdio": defaultStdioSource,
	"s3":    defaultS3Source,
})

type Reader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

type Source interface {
	NewReader(context.Context, URI) (Reader, error)
	NewWriter(context.Context, URI) (io.WriteCloser, error)
	ReadFile(context.Context, URI) ([]byte, error)
	WriteFile(context.Context, []byte, URI) error
	WriteFileIfNotExists(context.Context, []byte, URI) error
	Remove(context.Context, URI) error
	RemoveAll(context.Context, URI) error
	// Exists returns true if the specified uri exists and an error is there
	// was an error finding this information.
	Exists(context.Context, URI) (bool, error)
	Stat(context.Context, URI) (Info, error)
	ReadDir(context.Context, URI) ([]Info, error)
}

type Info interface {
	Name() string
	Size() int64
	ModTime() time.Time
	IsDir() bool
}

type DirMaker interface {
	MkdirAll(URI, os.FileMode) error
}

type Replacer interface {
	io.WriteCloser
	Abort()
}

// A ReplacerAble source supports atomic updates to a URI.
type ReplacerAble interface {
	NewReplacer(context.Context, URI) (Replacer, error)
}

func NewReader(ctx context.Context, uri URI) (Reader, error) {
	return DefaultMuxSource.NewReader(ctx, uri)
}

func NewWriter(ctx context.Context, uri URI) (io.WriteCloser, error) {
	return DefaultMuxSource.NewWriter(ctx, uri)
}

func ReadFile(ctx context.Context, uri URI) ([]byte, error) {
	return DefaultMuxSource.ReadFile(ctx, uri)
}

func WriteFile(ctx context.Context, uri URI, d []byte) error {
	return DefaultMuxSource.WriteFile(ctx, uri, d)
}

func WriteFileIfNotExists(ctx context.Context, uri URI, d []byte) error {
	return DefaultMuxSource.WriteFileIfNotExists(ctx, uri, d)
}

func Exists(ctx context.Context, uri URI) (bool, error) {
	return DefaultMuxSource.Exists(ctx, uri)
}

func Remove(ctx context.Context, uri URI) error {
	return DefaultMuxSource.Remove(ctx, uri)
}

func RemoveAll(ctx context.Context, uri URI) error {
	return DefaultMuxSource.RemoveAll(ctx, uri)
}

func Stat(ctx context.Context, uri URI) (Info, error) {
	return DefaultMuxSource.Stat(ctx, uri)
}

func ReadDir(ctx context.Context, uri URI) ([]Info, error) {
	return DefaultMuxSource.ReadDir(ctx, uri)
}

func GetSource(uri URI) (Source, error) {
	return DefaultMuxSource.GetSource(uri)
}

func Replace(ctx context.Context, uri URI, fn func(w io.Writer) error) error {
	return DefaultMuxSource.Replace(ctx, uri, fn)
}

func MkdirAll(uri URI, mode os.FileMode) error {
	return DefaultMuxSource.MkdirAll(uri, mode)
}
