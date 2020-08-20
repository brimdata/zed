//go:generate mockgen -destination=./mock/mock_source.go -package=mock github.com/brimsec/zq/pkg/iosrc Source

package iosrc

import (
	"fmt"
	"io"
	"os"
	"time"
)

const FileScheme = "file"

var schemes = map[string]Source{
	"file":  DefaultFileSource,
	"stdio": defaultStdioSource,
	"s3":    defaultS3Source,
}

type Reader interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

type Source interface {
	NewReader(URI) (Reader, error)
	NewWriter(URI) (io.WriteCloser, error)
	ReadFile(URI) ([]byte, error)
	WriteFile([]byte, URI) error
	Remove(URI) error
	RemoveAll(URI) error
	// Exists returns true if the specified uri exists and an error is there
	// was an error finding this information.
	Exists(URI) (bool, error)
	Stat(URI) (Info, error)
}

type Info interface {
	Size() int64
	ModTime() time.Time
}

type DirMaker interface {
	MkdirAll(URI, os.FileMode) error
}

// A ReplacerAble source supports atomic updates to a URI.
type ReplacerAble interface {
	NewReplacer(URI) (io.WriteCloser, error)
}

func NewReader(uri URI) (Reader, error) {
	source, err := GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.NewReader(uri)
}

func NewWriter(uri URI) (io.WriteCloser, error) {
	source, err := GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.NewWriter(uri)
}

func ReadFile(uri URI) ([]byte, error) {
	source, err := GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.ReadFile(uri)
}

func WriteFile(uri URI, d []byte) error {
	source, err := GetSource(uri)
	if err != nil {
		return err
	}
	return source.WriteFile(d, uri)
}

func Exists(uri URI) (bool, error) {
	source, err := GetSource(uri)
	if err != nil {
		return false, err
	}
	return source.Exists(uri)
}

func Remove(uri URI) error {
	source, err := GetSource(uri)
	if err != nil {
		return err
	}
	return source.Remove(uri)
}

func RemoveAll(uri URI) error {
	source, err := GetSource(uri)
	if err != nil {
		return err
	}
	return source.RemoveAll(uri)
}

func GetSource(uri URI) (Source, error) {
	scheme := getScheme(uri)
	source, ok := schemes[scheme]
	if !ok {
		return nil, fmt.Errorf("unknown scheme: %q", scheme)
	}
	return source, nil
}

func Stat(uri URI) (Info, error) {
	source, err := GetSource(uri)
	if err != nil {
		return nil, err
	}
	return source.Stat(uri)
}

func getScheme(uri URI) string {
	if uri.Scheme == "" {
		return FileScheme
	}
	return uri.Scheme
}
